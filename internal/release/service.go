package release

import (
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"context-rail/internal/change"
	"context-rail/internal/topology"
)

// ChangeReader gives the release gate the current view of a Change.
type ChangeReader interface {
	Get(projectID, changeID string) (*change.View, error)
}

// TopologyReader gives the current environment topology.
type TopologyReader interface {
	Get(projectID string) (*topology.State, error)
}

// Service applies the governed promotion path.
type Service struct {
	store    Store
	changes  ChangeReader
	topology TopologyReader
	now      func() time.Time
	mu       sync.Mutex
}

func NewService(store Store, changes ChangeReader, topologyReader TopologyReader) *Service {
	return &Service{store: store, changes: changes, topology: topologyReader, now: func() time.Time { return time.Now().UTC() }}
}

func (service *Service) WithClock(now func() time.Time) *Service {
	service.now = now
	return service
}

// Mutation envelope.
type Mutation struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

type CreateRequest struct {
	Mutation
	ChangeIDs           []string `json:"change_ids"`
	TargetEnvironmentID string   `json:"target_environment_id"`
	// SourceReleaseID turns the release into a promotion of an already
	// PROMOTED release: same digest, next environment, delta bound by approval.
	SourceReleaseID string `json:"source_release_id,omitempty"`
}

type BuildRequest struct {
	Mutation
	ImageDigest     string   `json:"image_digest"`
	ImageRef        string   `json:"image_ref"`
	BuildID         string   `json:"build_id"`
	SourceCommit    string   `json:"source_commit"`
	IncludesCommits []string `json:"includes_commits"`
	EvidenceRef     string   `json:"evidence_ref"`
}

type ApprovalRequest struct {
	Mutation
	Decision  string `json:"decision"` // APPROVE or REJECT
	Role      string `json:"role"`
	Rationale string `json:"rationale"`
}

type DeploymentRequest struct {
	Mutation
	IdempotencyKey     string       `json:"idempotency_key"`
	OperationID        string       `json:"operation_id"`
	Source             string       `json:"source"`
	Revision           string       `json:"revision"`
	ServiceURL         string       `json:"service_url"`
	DeployedDigest     string       `json:"deployed_digest"`
	DeployedTargetRef  string       `json:"deployed_target_ref"`
	DeployedConfigHash string       `json:"deployed_config_hash"`
	DeployedAt         string       `json:"deployed_at"`
	Smoke              *SmokeResult `json:"smoke"`
}

// List returns every release with live gates.
func (service *Service) List(projectID string) ([]View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	views := make([]View, 0, len(state.Releases))
	for index := range state.Releases {
		views = append(views, service.view(projectID, state, &state.Releases[index]))
	}
	return views, nil
}

// Get returns one release with live gates.
func (service *Service) Get(projectID, releaseID string) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	release := findRelease(state, releaseID)
	if release == nil {
		return nil, newError(CodeReleaseNotFound, "release %q is not in the ledger of %s", releaseID, projectID)
	}
	view := service.view(projectID, state, release)
	return &view, nil
}

// Create bundles accepted candidates into a release and evaluates the
// promotion gate. A bundle with one failing Change is blocked as a whole;
// each Change's reason is reported (UI-13). With source_release_id the
// release is a promotion: it inherits the digest and the change lineage of
// the source receipt and binds the environment delta instead (VS-007).
func (service *Service) Create(projectID string, request CreateRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, newError(CodeReasonRequired, "a reason is required to open a release")
	}
	target := strings.TrimSpace(request.TargetEnvironmentID)
	if target == "" {
		return nil, newError(CodeInvalidRequest, "a release needs a target_environment_id")
	}
	ids := uniqueTrimmed(request.ChangeIDs)
	at := service.stamp()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	release := Release{
		ReleaseID: fmt.Sprintf("REL-%03d", state.NextRelease), ProjectID: projectID,
		Manifest: Manifest{ReleaseID: fmt.Sprintf("REL-%03d", state.NextRelease), ProjectID: projectID, PolicyVersion: PolicyVersion, Environment: EnvironmentConfig{EnvironmentID: target}},
		Reason:   request.Reason, CreatedBy: actor, CreatedAt: at, UpdatedAt: at, Deployments: []DeploymentRecord{},
	}
	if sourceID := strings.TrimSpace(request.SourceReleaseID); sourceID != "" {
		source := findRelease(state, sourceID)
		switch {
		case source == nil:
			return nil, newError(CodeSourceRelease, "source release %q is not in the ledger of %s", sourceID, projectID)
		case source.Status != StatusPromoted || source.Receipt == nil || source.Manifest.Build == nil:
			return nil, newError(CodeSourceRelease, "source release %s is %s; only a PROMOTED release with a receipt can be promoted further", sourceID, source.Status)
		case source.Manifest.Environment.EnvironmentID == target:
			return nil, newError(CodeSourceRelease, "source release %s already targets %s; a promotion moves the same digest to the next environment", sourceID, target)
		}
		sourceIDs := make([]string, 0, len(source.Manifest.Changes))
		for _, entry := range source.Manifest.Changes {
			sourceIDs = append(sourceIDs, entry.ChangeID)
		}
		if len(ids) > 0 && strings.Join(ids, ",") != strings.Join(sourceIDs, ",") {
			return nil, newError(CodeInvalidRequest, "a promotion carries exactly the changes of its source receipt %s (%s); it cannot add or drop changes", source.Receipt.ReceiptID, strings.Join(sourceIDs, ", "))
		}
		build := *source.Manifest.Build
		release.Manifest.Build = &build
		release.Manifest.Promotion = &PromotionSource{
			SourceReleaseID: source.ReleaseID, SourceReceiptID: source.Receipt.ReceiptID, SourceReceiptHash: source.Receipt.ReceiptHash,
			SourceRevision: source.Receipt.Deployment.Revision, SourceServiceURL: source.Receipt.Deployment.ServiceURL, SourceEnvironment: source.Receipt.Environment,
		}
		release.Manifest.Changes = append([]ReleaseChange{}, source.Manifest.Changes...)
	} else if len(ids) == 0 {
		return nil, newError(CodeInvalidRequest, "a release needs at least one change_id")
	} else {
		for _, id := range ids {
			release.Manifest.Changes = append(release.Manifest.Changes, ReleaseChange{ChangeID: id})
		}
	}
	state.NextRelease++
	result := service.evaluate(projectID, state, &release)
	release.Manifest.Changes = result.changes
	release.Manifest.Environment = result.environment
	release.Manifest.SourceEnvironment = result.source
	release.Manifest.Transition = result.transition
	release.Manifest.EnvironmentDelta = result.delta
	if release.Manifest.IsPromotion() {
		release.Manifest.EnvironmentDeltaHash = hashJSON(result.delta)
	}
	release.Manifest.ManifestHash = manifestHash(release.Manifest)
	release.Gates = service.regate(projectID, state, &release)
	release.Verdict, release.Status = verdictOf(release.Gates, nil, nil)
	state.Releases = append(state.Releases, release)
	service.audit(state, at, actor, "release.create", release.ReleaseID, fmt.Sprintf("%s: %s", release.Status, request.Reason))
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(projectID, state, &state.Releases[len(state.Releases)-1])
	return &view, nil
}

// RecordBuild attaches the observed image build. The build must cover every
// Change's accepted head commit; a digest is the release identity from here on.
func (service *Service) RecordBuild(projectID, releaseID string, request BuildRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	release := findRelease(state, releaseID)
	if release == nil {
		return nil, newError(CodeReleaseNotFound, "release %q is not in the ledger of %s", releaseID, projectID)
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, newError(CodeReasonRequired, "a reason is required to record a build")
	}
	if release.Status == StatusPromoted || release.Status == StatusRejected {
		return nil, newError(CodeReleaseClosed, "release %s is %s", releaseID, release.Status)
	}
	if release.Manifest.IsPromotion() {
		return nil, newError(CodeBuildInherited, "release %s promotes image %s from %s (%s); a promotion never rebuilds — a different image needs a new %s release", releaseID, shortHash(release.Manifest.Build.ImageDigest), release.Manifest.Promotion.SourceReleaseID, release.Manifest.Promotion.SourceReceiptID, release.Manifest.Promotion.SourceEnvironment.EnvironmentID)
	}
	digest := strings.TrimSpace(request.ImageDigest)
	if !strings.HasPrefix(digest, "sha256:") || len(digest) < 20 {
		return nil, newError(CodeInvalidRequest, "image_digest must be a sha256: digest, not a mutable tag")
	}
	sourceCommit := strings.TrimSpace(request.SourceCommit)
	if sourceCommit == "" {
		return nil, newError(CodeInvalidRequest, "source_commit is required so the image can be tied to the accepted candidates")
	}
	at := service.stamp()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	service.syncManifest(projectID, state, release, at, actor)
	covered := map[string]bool{sourceCommit: true}
	for _, commit := range request.IncludesCommits {
		if trimmed := strings.TrimSpace(commit); trimmed != "" {
			covered[trimmed] = true
		}
	}
	for _, entry := range release.Manifest.Changes {
		if entry.HeadCommit != "" && !covered[entry.HeadCommit] {
			return nil, newError(CodeBuildMismatch, "build from %s does not include the accepted candidate commit %s of %s; the image is not the reviewed change", shortHash(sourceCommit), shortHash(entry.HeadCommit), entry.ChangeID)
		}
	}
	release.Manifest.Build = &BuildEvidence{
		ImageDigest: digest, ImageRef: strings.TrimSpace(request.ImageRef), BuildID: strings.TrimSpace(request.BuildID),
		SourceCommit: sourceCommit, IncludesCommits: uniqueTrimmed(request.IncludesCommits), EvidenceRef: strings.TrimSpace(request.EvidenceRef), ObservedAt: at,
	}
	previousHash := release.Manifest.ManifestHash
	release.Manifest.ManifestHash = manifestHash(release.Manifest)
	if release.Approval != nil && release.Approval.Decision == "APPROVED" && release.Approval.ManifestHash != release.Manifest.ManifestHash {
		// A new build after approval changes the identity: the approval no longer binds.
		release.Approval = nil
		service.audit(state, at, actor, "release.approval.invalidated", release.ReleaseID, "manifest changed by new build "+shortHash(previousHash)+" → "+shortHash(release.Manifest.ManifestHash))
	}
	release.Gates = service.regate(projectID, state, release)
	release.Verdict, release.Status = verdictOf(release.Gates, release.Approval, release.Receipt)
	release.UpdatedAt = at
	service.audit(state, at, actor, "release.build", release.ReleaseID, request.Reason)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(projectID, state, release)
	return &view, nil
}

// Approve records the third human decision, bound to the manifest hash.
func (service *Service) Approve(projectID, releaseID string, request ApprovalRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	release := findRelease(state, releaseID)
	if release == nil {
		return nil, newError(CodeReleaseNotFound, "release %q is not in the ledger of %s", releaseID, projectID)
	}
	actor := strings.TrimSpace(request.Actor)
	if actor == "" {
		return nil, newError(CodeActorRequired, "a release approval must name its actor")
	}
	rationale := firstNonEmpty(strings.TrimSpace(request.Rationale), strings.TrimSpace(request.Reason))
	if rationale == "" {
		return nil, newError(CodeReasonRequired, "a rationale is required for a release decision")
	}
	if release.Status == StatusPromoted || release.Status == StatusRejected {
		return nil, newError(CodeReleaseClosed, "release %s is %s", releaseID, release.Status)
	}
	at := service.stamp()
	decision := strings.ToUpper(strings.TrimSpace(request.Decision))
	switch decision {
	case "REJECT", "REJECTED":
		release.Approval = &Approval{Actor: actor, Role: firstNonEmpty(strings.TrimSpace(request.Role), "undeclared"), At: at, Decision: "REJECTED", Reason: rationale, ManifestHash: release.Manifest.ManifestHash}
		release.Status = StatusRejected
		release.Verdict = StatusRejected
	case "APPROVE", "APPROVED", "ACCEPT":
		service.syncManifest(projectID, state, release, at, actor)
		release.Gates = service.regate(projectID, state, release)
		if blocked := firstBlocked(release.Gates); blocked != nil {
			return nil, newError(CodeNotApprovable, "release %s cannot be approved: %s — %s", releaseID, blocked.Gate, blocked.Detail)
		}
		if release.Manifest.Build == nil {
			return nil, newError(CodeNotApprovable, "release %s has no build evidence; an approval must bind an image digest", releaseID)
		}
		if release.Approval != nil && release.Approval.Decision == "APPROVED" && release.Approval.ManifestHash == release.Manifest.ManifestHash {
			return nil, newError(CodeAlreadyApproved, "release %s is already approved for manifest %s", releaseID, shortHash(release.Manifest.ManifestHash))
		}
		waiver := ""
		for _, entry := range release.Manifest.Changes {
			if entry.RunStartedBy != "" && entry.RunStartedBy == actor {
				if !policyAllowsSingleOperator(release.Manifest.Environment) {
					return nil, newError(CodeApproverConflict, "%s started the agent run for %s and cannot also approve its release; the target policy has no single-operator controls", actor, entry.ChangeID)
				}
				waiver = "approver also started run " + entry.RunID + "; accepted under documented single-operator controls of " + release.Manifest.Environment.EnvironmentID
			}
		}
		release.Approval = &Approval{
			Actor: actor, Role: firstNonEmpty(strings.TrimSpace(request.Role), "undeclared"), At: at, Decision: "APPROVED", Reason: rationale,
			ManifestHash: release.Manifest.ManifestHash, ExpiresAt: service.now().Add(ApprovalTTLHours * time.Hour).Format(time.RFC3339), Waiver: waiver,
		}
		release.Verdict, release.Status = verdictOf(release.Gates, release.Approval, release.Receipt)
	default:
		return nil, newError(CodeInvalidRequest, "decision must be APPROVE or REJECT")
	}
	release.UpdatedAt = at
	service.audit(state, at, actor, "release."+strings.ToLower(release.Approval.Decision), release.ReleaseID, rationale)
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(projectID, state, release)
	return &view, nil
}

// RecordDeployment verifies what was actually deployed against the approved
// manifest and the current topology, and issues the receipt. The same
// idempotency key replays the earlier result instead of recording twice.
func (service *Service) RecordDeployment(projectID, releaseID string, request DeploymentRequest) (*View, error) {
	service.mu.Lock()
	defer service.mu.Unlock()
	state, err := service.load(projectID)
	if err != nil {
		return nil, err
	}
	release := findRelease(state, releaseID)
	if release == nil {
		return nil, newError(CodeReleaseNotFound, "release %q is not in the ledger of %s", releaseID, projectID)
	}
	key := strings.TrimSpace(request.IdempotencyKey)
	if key == "" {
		return nil, newError(CodeInvalidRequest, "idempotency_key is required so a retried deploy cannot be recorded twice")
	}
	for index := range release.Deployments {
		if release.Deployments[index].IdempotencyKey == key {
			// Replay: same key → same answer, no new attempt, no new receipt.
			view := service.view(projectID, state, release)
			return &view, nil
		}
	}
	if strings.TrimSpace(request.Reason) == "" {
		return nil, newError(CodeReasonRequired, "a reason is required to record a deployment")
	}
	if release.Status == StatusPromoted {
		return nil, newError(CodeAlreadyPromoted, "release %s is already PROMOTED (receipt %s); a new deployment needs a new release", releaseID, release.Receipt.ReceiptID)
	}
	if release.Status == StatusRejected {
		return nil, newError(CodeReleaseClosed, "release %s was REJECTED", releaseID)
	}
	at := service.stamp()
	actor := firstNonEmpty(strings.TrimSpace(request.Actor), "local-operator")
	record := DeploymentRecord{
		AttemptID: fmt.Sprintf("%s-D%02d", release.ReleaseID, len(release.Deployments)+1), IdempotencyKey: key,
		OperationID: strings.TrimSpace(request.OperationID), Source: firstNonEmpty(strings.TrimSpace(request.Source), "declared"),
		Revision: strings.TrimSpace(request.Revision), ServiceURL: strings.TrimSpace(request.ServiceURL),
		DeployedDigest: strings.TrimSpace(request.DeployedDigest), DeployedTargetRef: strings.TrimSpace(request.DeployedTargetRef),
		DeployedConfigHash: strings.TrimSpace(request.DeployedConfigHash), DeployedAt: firstNonEmpty(strings.TrimSpace(request.DeployedAt), at),
		DeployedBy: actor, Smoke: request.Smoke, RecordedAt: at,
	}
	service.syncManifest(projectID, state, release, at, actor)
	record.Gates = service.deploymentGates(projectID, state, release, record)
	if blocked := firstNotPassing(record.Gates); blocked != nil {
		// Every deployment gate must PASS: missing evidence is not a pass.
		record.Outcome = StatusPromotionFailed
		release.Status = StatusPromotionFailed
		release.Verdict = StatusPromotionFailed
		for _, gate := range record.Gates {
			if (gate.Gate == "target_config_drift" || gate.Gate == "config_drift") && (gate.Status == GateBlocked || gate.Status == GateStale) && release.Approval != nil && release.Approval.Decision == "APPROVED" {
				// UI-14: the environment moved after approval; the approval no longer binds.
				release.Approval.Decision = "STALE"
				release.Approval.Reason = release.Approval.Reason + " [invalidated: " + gate.Detail + "]"
				service.audit(state, at, actor, "release.approval.invalidated", release.ReleaseID, gate.Detail)
			}
		}
	} else {
		record.Outcome = StatusPromoted
		release.Status = StatusPromoted
		release.Verdict = StatusPromoted
	}
	release.Deployments = append(release.Deployments, record)
	receipt := service.receipt(state, release, record)
	release.Receipt = &receipt
	release.UpdatedAt = at
	service.audit(state, at, actor, "release.deployment", record.AttemptID, fmt.Sprintf("%s: %s", record.Outcome, request.Reason))
	if err := service.store.Save(projectID, state); err != nil {
		return nil, err
	}
	view := service.view(projectID, state, release)
	return &view, nil
}

// --- gates -----------------------------------------------------------------------

// bundleResult is what evaluating a release against the live ledger yields:
// the manifest inputs plus the pre-build gate results.
type bundleResult struct {
	changes     []ReleaseChange
	environment EnvironmentConfig
	source      string
	transition  string
	delta       []EnvironmentDeltaField
	gates       []GateResult
}

// executableTypes are the standard environment types P0 executes releases
// against. Every other node is verified by evidence and gates only.
var executableTypes = map[string]bool{"staging": true, "prod-demo": true}

// evaluate resolves every Change and the target environment of a release
// against the live ledger. For a promotion (VS-007) it also checks the source
// receipt, the topology order and the change lineage frozen by that receipt.
func (service *Service) evaluate(projectID string, state *State, release *Release) bundleResult {
	target := release.Manifest.Environment.EnvironmentID
	promotion := release.Manifest.Promotion
	result := bundleResult{gates: []GateResult{}, environment: EnvironmentConfig{EnvironmentID: target}}

	topologyState, err := service.topology.Get(projectID)
	var ordered []topology.Environment
	var current *topology.Version
	switch {
	case err != nil:
		result.gates = append(result.gates, GateResult{Gate: "environment", Status: GateBlocked, Detail: "topology unavailable: " + err.Error()})
	case topologyState == nil || topologyState.Current() == nil:
		result.gates = append(result.gates, GateResult{Gate: "environment", Status: GateBlocked, Detail: "no topology version to bind the release to"})
	default:
		current = topologyState.Current()
		ordered = append([]topology.Environment(nil), current.Environments...)
		sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].Sequence < ordered[j].Sequence })
		index := -1
		for position, env := range ordered {
			if env.ID == target {
				index = position
			}
		}
		switch {
		case index < 0:
			result.gates = append(result.gates, GateResult{Gate: "environment", Status: GateBlocked, Detail: fmt.Sprintf("environment %q is not in topology v%d", target, current.Version)})
		case ordered[index].Type == "production":
			result.gates = append(result.gates, GateResult{Gate: "environment", Status: GateBlocked, Detail: "production is READ_ONLY/BLOCKED in P0; releases execute only for staging or an isolated prod-demo"})
		case !executableTypes[ordered[index].Type]:
			result.gates = append(result.gates, GateResult{Gate: "environment", Status: GateBlocked, Detail: fmt.Sprintf("%s is a %s environment; P0 executes releases only for staging or an isolated prod-demo — other environments are verified by evidence gates, not deployed by a release", target, ordered[index].Type)})
		case ordered[index].Status != topology.StatusActive:
			result.gates = append(result.gates, GateResult{Gate: "environment", Status: GateBlocked, Detail: fmt.Sprintf("environment %s is %s, not ACTIVE", target, ordered[index].Status)})
		default:
			result.environment = environmentConfig(ordered[index], current.Version)
			for prev := index - 1; prev >= 0; prev-- {
				if ordered[prev].Status != topology.StatusRetired {
					result.source = ordered[prev].ID
					break
				}
			}
			result.gates = append(result.gates, GateResult{Gate: "environment", Status: GatePass, Detail: fmt.Sprintf("%s (%s) at %s, topology v%d, config %s", target, ordered[index].Type, ordered[index].TargetRef, current.Version, shortHash(result.environment.ConfigHash))})
		}
	}
	if promotion != nil {
		result.source = promotion.SourceEnvironment.EnvironmentID
		result.gates = append(result.gates, service.promotionGates(state, release, ordered)...)
		result.delta = environmentDelta(promotion.SourceEnvironment, result.environment)
	}
	result.transition = firstNonEmpty(result.source, "entry") + "-to-" + target

	// Per-change lineage. A promotion keeps the lineage its source receipt
	// froze; the live ledger must still agree with it.
	var frozen map[string]ReleaseChange
	if promotion != nil {
		frozen = map[string]ReleaseChange{}
		for _, entry := range release.Manifest.Changes {
			frozen[entry.ChangeID] = entry
		}
	}
	for _, listed := range release.Manifest.Changes {
		view, err := service.changes.Get(projectID, listed.ChangeID)
		if err != nil {
			result.gates = append(result.gates, GateResult{Gate: "change", ChangeID: listed.ChangeID, Status: GateBlocked, Detail: "cannot read change: " + err.Error()})
			result.changes = append(result.changes, ReleaseChange{ChangeID: listed.ChangeID})
			continue
		}
		entry, gate := releaseChangeOf(view, target, promotion)
		if promotion != nil {
			if reference, ok := frozen[listed.ChangeID]; ok && gate.Status == GatePass && lineageHash([]ReleaseChange{entry}) != lineageHash([]ReleaseChange{reference}) {
				gate.Status = GateStale
				gate.Detail = fmt.Sprintf("lineage changed since %s: the receipt bound %s @ %s (%s v%d) but the ledger now has %s @ %s (%s v%d); the promoted image no longer represents the accepted change", promotion.SourceReceiptID, reference.CandidateID, shortHash(reference.HeadCommit), reference.DecisionID, reference.DecisionVersion, entry.CandidateID, shortHash(entry.HeadCommit), entry.DecisionID, entry.DecisionVersion)
			}
			// The manifest of a promotion keeps the frozen lineage, whatever the ledger says now.
			entry = frozen[listed.ChangeID]
		}
		result.changes = append(result.changes, entry)
		result.gates = append(result.gates, gate)
	}
	return result
}

// promotionGates checks the source receipt and the topology order of a
// promotion: the source must still be PROMOTED with an intact receipt, the
// same digest must not already have been promoted to this target, and the
// target must be the next environment after the source.
func (service *Service) promotionGates(state *State, release *Release, ordered []topology.Environment) []GateResult {
	promotion := release.Manifest.Promotion
	target := release.Manifest.Environment.EnvironmentID
	gates := []GateResult{}
	source := findRelease(state, promotion.SourceReleaseID)
	switch {
	case source == nil:
		gates = append(gates, GateResult{Gate: "source_release", Status: GateBlocked, Detail: "source release " + promotion.SourceReleaseID + " is no longer in the ledger"})
	case source.Status != StatusPromoted || source.Receipt == nil:
		gates = append(gates, GateResult{Gate: "source_release", Status: GateBlocked, Detail: fmt.Sprintf("source release %s is %s, not PROMOTED", source.ReleaseID, source.Status)})
	case source.Receipt.ReceiptID != promotion.SourceReceiptID || source.Receipt.ReceiptHash != promotion.SourceReceiptHash:
		gates = append(gates, GateResult{Gate: "source_release", Status: GateBlocked, Detail: fmt.Sprintf("receipt %s of %s no longer matches the hash this promotion was cut from (%s)", promotion.SourceReceiptID, source.ReleaseID, shortHash(promotion.SourceReceiptHash))})
	case source.Manifest.Build == nil || release.Manifest.Build == nil || source.Manifest.Build.ImageDigest != release.Manifest.Build.ImageDigest:
		gates = append(gates, GateResult{Gate: "source_release", Status: GateBlocked, Detail: "the inherited digest differs from the source receipt's digest"})
	default:
		detail := fmt.Sprintf("%s PROMOTED to %s as revision %s, receipt %s (%s)", source.ReleaseID, promotion.SourceEnvironment.EnvironmentID, firstNonEmpty(promotion.SourceRevision, "∅"), promotion.SourceReceiptID, shortHash(promotion.SourceReceiptHash))
		status := GatePass
		for index := range state.Releases {
			other := &state.Releases[index]
			if other.ReleaseID == release.ReleaseID || other.Manifest.Promotion == nil || other.Status != StatusPromoted || other.Receipt == nil {
				continue
			}
			if other.Manifest.Promotion.SourceReleaseID == source.ReleaseID && other.Manifest.Environment.EnvironmentID == target {
				status = GateBlocked
				detail = fmt.Sprintf("%s was already promoted to %s by %s (%s); a target receives the same source receipt once", source.ReleaseID, target, other.ReleaseID, other.Receipt.ReceiptID)
				break
			}
		}
		gates = append(gates, GateResult{Gate: "source_release", Status: status, Detail: detail})
	}
	// Topology order: the target must be the next non-retired environment after the source.
	sourceIndex, targetIndex := -1, -1
	for position, env := range ordered {
		if env.ID == promotion.SourceEnvironment.EnvironmentID {
			sourceIndex = position
		}
		if env.ID == target {
			targetIndex = position
		}
	}
	switch {
	case len(ordered) == 0:
		// environment gate already reports the topology problem
	case sourceIndex < 0:
		gates = append(gates, GateResult{Gate: "promotion_order", Status: GateBlocked, Detail: "source environment " + promotion.SourceEnvironment.EnvironmentID + " is no longer in the topology"})
	case targetIndex < 0:
		// environment gate already reports it
	default:
		// The next executable node after the source is the only valid target;
		// evidence-only nodes in between (development, testing, uat, other)
		// are verified by gates, never deployed by a release.
		next, skipped := "", []string{}
		for position := sourceIndex + 1; position < len(ordered); position++ {
			if ordered[position].Status == topology.StatusRetired {
				continue
			}
			if executableTypes[ordered[position].Type] || ordered[position].Type == "production" {
				next = ordered[position].ID
				break
			}
			skipped = append(skipped, ordered[position].ID+" ("+ordered[position].Type+")")
		}
		via := ""
		if len(skipped) > 0 {
			via = " via evidence-only " + strings.Join(skipped, ", ")
		}
		if next != target {
			gates = append(gates, GateResult{Gate: "promotion_order", Status: GateBlocked, Detail: fmt.Sprintf("promotion must follow the topology order: after %s comes %s%s, not %s", promotion.SourceEnvironment.EnvironmentID, firstNonEmpty(next, "nothing"), via, target)})
		} else {
			gates = append(gates, GateResult{Gate: "promotion_order", Status: GatePass, Detail: fmt.Sprintf("%s → %s%s follows topology v%d", promotion.SourceEnvironment.EnvironmentID, target, via, release.Manifest.Environment.TopologyVersion)})
		}
	}
	return gates
}

// environmentDelta lists what differs between the environment a receipt was
// issued for and the promotion target. Same digest ≠ same environment.
func environmentDelta(from, to EnvironmentConfig) []EnvironmentDeltaField {
	delta := []EnvironmentDeltaField{}
	if from.Type != to.Type {
		delta = append(delta, EnvironmentDeltaField{Field: "type", From: from.Type, To: to.Type})
	}
	if from.TargetRef != to.TargetRef {
		delta = append(delta, EnvironmentDeltaField{Field: "target_ref", From: from.TargetRef, To: to.TargetRef})
	}
	if strings.Join(from.RequiredEvidence, ",") != strings.Join(to.RequiredEvidence, ",") {
		delta = append(delta, EnvironmentDeltaField{Field: "required_evidence", From: append([]string{}, from.RequiredEvidence...), To: append([]string{}, to.RequiredEvidence...)})
	}
	if from.ApproverPolicy != to.ApproverPolicy {
		delta = append(delta, EnvironmentDeltaField{Field: "approver_policy", From: from.ApproverPolicy, To: to.ApproverPolicy})
	}
	return delta
}

// releaseChangeOf extracts the lineage of a Change and judges whether it may
// be promoted: accepted candidate, current decision, matching target. For a
// promotion the decision must target the source environment the receipt
// covered, not the promotion target.
func releaseChangeOf(view *change.View, target string, promotion *PromotionSource) (ReleaseChange, GateResult) {
	entry := ReleaseChange{ChangeID: view.Change.ChangeID, Title: view.Change.Title}
	if view.Decision != nil {
		entry.DecisionID = view.Decision.DecisionID
		entry.DecisionVersion = view.Decision.Version
		entry.SourceSnapshotHash = view.Decision.SourceSnapshotHash
		entry.TargetEnvironment = view.Decision.TargetEnvironment.ID
	}
	if view.WorkOrder != nil {
		entry.WorkOrderID = view.WorkOrder.WorkOrderID
		entry.WorkOrderHash = view.WorkOrder.WorkOrderHash
	}
	var accepted *change.Candidate
	for index := range view.Change.Candidates {
		if view.Change.Candidates[index].Status == "ACCEPTED_FOR_PROMOTION" {
			accepted = &view.Change.Candidates[index]
		}
	}
	expectedTarget := target
	if promotion != nil {
		expectedTarget = promotion.SourceEnvironment.EnvironmentID
	}
	gate := GateResult{Gate: "change", ChangeID: view.Change.ChangeID}
	switch {
	case view.Decision == nil || view.Decision.Status != "ACCEPTED_FOR_DEVELOPMENT":
		gate.Status, gate.Detail = GateBlocked, "no accepted DecisionRecord"
	case view.Staleness.Stale:
		gate.Status, gate.Detail = GateStale, "decision is STALE: "+view.Staleness.Reason
	case view.WorkOrder == nil || view.WorkOrder.Status != "ISSUED":
		gate.Status, gate.Detail = GateBlocked, "no issued Work Order"
	case accepted == nil:
		gate.Status, gate.Detail = GateBlocked, "no candidate accepted for promotion (the second human decision is missing)"
	case view.Decision.TargetEnvironment.ID != expectedTarget:
		gate.Status, gate.Detail = GateBlocked, fmt.Sprintf("decision targets %s, release targets %s", view.Decision.TargetEnvironment.ID, expectedTarget)
	default:
		gate.Status, gate.Detail = GatePass, fmt.Sprintf("%s v%d · %s · %s @ %s accepted by %s", view.Decision.DecisionID, view.Decision.Version, view.WorkOrder.WorkOrderID, accepted.CandidateID, shortHash(accepted.Observation.HeadCommit), accepted.HumanDecision.Actor)
	}
	if accepted != nil {
		entry.CandidateID = accepted.CandidateID
		entry.RunID = accepted.Run.RunID
		entry.RunStartedBy = accepted.Run.StartedBy
		entry.Branch = accepted.Observation.Branch
		entry.HeadCommit = accepted.Observation.HeadCommit
		entry.CandidateAcceptedBy = accepted.HumanDecision.Actor
		for _, result := range accepted.Gates {
			if result.Gate == "independent_review" {
				entry.ReviewGate = result.Status
			}
		}
	}
	return entry, gate
}

// regate re-evaluates the pre-deploy gates against the live change and
// topology state, keeping the manifest's bound configuration for drift checks.
func (service *Service) regate(projectID string, state *State, release *Release) []GateResult {
	result := service.evaluate(projectID, state, release)
	gates := result.gates
	if !release.Manifest.IsPromotion() && release.Approval != nil && release.Approval.Decision == "APPROVED" && lineageHash(result.changes) != lineageHash(release.Manifest.Changes) {
		gates = append(gates, GateResult{Gate: "manifest_drift", Status: GateStale, Detail: "the accepted candidates behind this release changed after approval (new candidate, decision or work order); the approval binds a different manifest"})
	}
	if release.Manifest.Environment.ConfigHash != "" && result.environment.ConfigHash != "" && result.environment.ConfigHash != release.Manifest.Environment.ConfigHash {
		gates = append(gates, GateResult{Gate: "target_config_drift", Status: GateStale, Detail: fmt.Sprintf("target %s configuration changed after the manifest was bound (topology v%d %s → v%d %s); re-evaluate or re-approve", release.Manifest.Environment.EnvironmentID, release.Manifest.Environment.TopologyVersion, shortHash(release.Manifest.Environment.ConfigHash), result.environment.TopologyVersion, shortHash(result.environment.ConfigHash))})
	}
	if release.Manifest.IsPromotion() {
		if len(release.Manifest.EnvironmentDelta) == 0 {
			gates = append(gates, GateResult{Gate: "environment_delta", Status: GatePass, Detail: "target configuration equals the source environment; nothing but the target changes"})
		} else {
			fields := make([]string, 0, len(release.Manifest.EnvironmentDelta))
			for _, field := range release.Manifest.EnvironmentDelta {
				fields = append(fields, field.Field)
			}
			gates = append(gates, GateResult{Gate: "environment_delta", Status: GatePass, Detail: fmt.Sprintf("%d field(s) differ from %s: %s — delta %s is bound by the approval", len(fields), release.Manifest.Promotion.SourceEnvironment.EnvironmentID, strings.Join(fields, ", "), shortHash(release.Manifest.EnvironmentDeltaHash))})
		}
	}
	if release.Manifest.Build == nil {
		gates = append(gates, GateResult{Gate: "build", Status: GateNeedsEvidence, Detail: "no image build recorded; the release has no digest identity yet"})
	} else if release.Manifest.IsPromotion() {
		gates = append(gates, GateResult{Gate: "build", Status: GatePass, Detail: "image " + shortHash(release.Manifest.Build.ImageDigest) + " inherited from " + release.Manifest.Promotion.SourceReleaseID + " / " + release.Manifest.Promotion.SourceReceiptID + "; a promotion never rebuilds"})
	} else {
		gates = append(gates, GateResult{Gate: "build", Status: GatePass, Detail: "image " + shortHash(release.Manifest.Build.ImageDigest) + " from " + shortHash(release.Manifest.Build.SourceCommit)})
	}
	return gates
}

// syncManifest refreshes the per-change lineage from the live ledger before a
// mutation, so the manifest (and the hash an approval binds) always names the
// candidates that are actually accepted now. The bound environment config is
// never refreshed: drift against it is what UI-14 reports. A promotion keeps
// the lineage its source receipt froze; a changed ledger makes it STALE instead.
func (service *Service) syncManifest(projectID string, state *State, release *Release, at, actor string) {
	if release.Status == StatusPromoted || release.Status == StatusRejected || release.Manifest.IsPromotion() {
		return
	}
	result := service.evaluate(projectID, state, release)
	if lineageHash(result.changes) == lineageHash(release.Manifest.Changes) {
		return
	}
	previous := release.Manifest.ManifestHash
	release.Manifest.Changes = result.changes
	release.Manifest.ManifestHash = manifestHash(release.Manifest)
	if release.Approval != nil && release.Approval.Decision == "APPROVED" && release.Approval.ManifestHash != release.Manifest.ManifestHash {
		release.Approval.Decision = "STALE"
		release.Approval.Reason = release.Approval.Reason + " [invalidated: manifest " + shortHash(previous) + " → " + shortHash(release.Manifest.ManifestHash) + "]"
		service.audit(state, at, actor, "release.approval.invalidated", release.ReleaseID, "candidate lineage changed after approval")
	}
}

func lineageHash(changes []ReleaseChange) string { return hashJSON(changes) }

// deploymentGates checks the observed deployment against the approved manifest.
func (service *Service) deploymentGates(projectID string, state *State, release *Release, record DeploymentRecord) []GateResult {
	gates := service.regate(projectID, state, release)
	approval := release.Approval
	switch {
	case approval == nil || approval.Decision != "APPROVED":
		gates = append(gates, GateResult{Gate: "release_approval", Status: GateBlocked, Detail: "no valid human release approval"})
	case approval.ManifestHash != release.Manifest.ManifestHash:
		gates = append(gates, GateResult{Gate: "release_approval", Status: GateBlocked, Detail: "approval was given for manifest " + shortHash(approval.ManifestHash) + " but the manifest is now " + shortHash(release.Manifest.ManifestHash)})
	default:
		if expires, err := time.Parse(time.RFC3339, approval.ExpiresAt); err == nil && service.now().After(expires) {
			gates = append(gates, GateResult{Gate: "release_approval", Status: GateBlocked, Detail: "approval expired at " + approval.ExpiresAt})
		} else {
			gates = append(gates, GateResult{Gate: "release_approval", Status: GatePass, Detail: "approved by " + approval.Actor + " for manifest " + shortHash(approval.ManifestHash)})
		}
	}
	// Digest drift: same digest is necessary (not sufficient) for promotion.
	switch {
	case release.Manifest.Build == nil:
		gates = append(gates, GateResult{Gate: "digest_drift", Status: GateBlocked, Detail: "no approved digest to compare against"})
	case record.DeployedDigest == "":
		gates = append(gates, GateResult{Gate: "digest_drift", Status: GateNeedsEvidence, Detail: "deployed image digest not observed"})
	case record.DeployedDigest != release.Manifest.Build.ImageDigest:
		gates = append(gates, GateResult{Gate: "digest_drift", Status: GateBlocked, Detail: "deployed " + shortHash(record.DeployedDigest) + " ≠ approved " + shortHash(release.Manifest.Build.ImageDigest)})
	default:
		gates = append(gates, GateResult{Gate: "digest_drift", Status: GatePass, Detail: "deployed digest equals the approved digest"})
	}
	// Target drift: the revision must land on the environment's declared target.
	switch {
	case record.DeployedTargetRef == "":
		gates = append(gates, GateResult{Gate: "target_drift", Status: GateNeedsEvidence, Detail: "deployed target not observed"})
	case record.DeployedTargetRef != release.Manifest.Environment.TargetRef:
		gates = append(gates, GateResult{Gate: "target_drift", Status: GateBlocked, Detail: "deployed to " + record.DeployedTargetRef + " but the approved target is " + release.Manifest.Environment.TargetRef})
	default:
		gates = append(gates, GateResult{Gate: "target_drift", Status: GatePass, Detail: "deployed to the approved target " + record.DeployedTargetRef})
	}
	if record.DeployedConfigHash != "" && record.DeployedConfigHash != release.Manifest.Environment.ConfigHash {
		gates = append(gates, GateResult{Gate: "config_drift", Status: GateBlocked, Detail: "deployed configuration " + shortHash(record.DeployedConfigHash) + " ≠ approved " + shortHash(release.Manifest.Environment.ConfigHash)})
	}
	if record.Revision == "" {
		gates = append(gates, GateResult{Gate: "revision_observed", Status: GateNeedsEvidence, Detail: "no revision name observed; the deployment cannot be re-read"})
	} else {
		gates = append(gates, GateResult{Gate: "revision_observed", Status: GatePass, Detail: record.Revision})
	}
	if promotion := release.Manifest.Promotion; promotion != nil && record.Revision != "" {
		// Cloud Run revisions belong to one service: a promotion creates a new
		// revision on the target; the source revision cannot be moved.
		if record.Revision == promotion.SourceRevision || (record.DeployedTargetRef != "" && record.DeployedTargetRef == promotion.SourceEnvironment.TargetRef) {
			gates = append(gates, GateResult{Gate: "new_revision", Status: GateBlocked, Detail: fmt.Sprintf("revision %s on %s is the %s deployment behind %s; a promotion creates a new revision on %s, revisions cannot move between services", record.Revision, firstNonEmpty(record.DeployedTargetRef, "?"), promotion.SourceEnvironment.EnvironmentID, promotion.SourceReceiptID, release.Manifest.Environment.EnvironmentID)})
		} else {
			gates = append(gates, GateResult{Gate: "new_revision", Status: GatePass, Detail: fmt.Sprintf("new revision %s on %s (source revision %s stays on %s)", record.Revision, release.Manifest.Environment.EnvironmentID, firstNonEmpty(promotion.SourceRevision, "∅"), promotion.SourceEnvironment.EnvironmentID)})
		}
	}
	switch {
	case record.Smoke == nil:
		gates = append(gates, GateResult{Gate: "smoke", Status: GateNeedsEvidence, Detail: "no smoke result; the deployment is not verified"})
	case strings.ToUpper(record.Smoke.Status) != "PASS":
		gates = append(gates, GateResult{Gate: "smoke", Status: GateBlocked, Detail: "smoke " + record.Smoke.Status + " (" + firstNonEmpty(record.Smoke.EvidenceRef, "no evidence ref") + ")"})
	default:
		gates = append(gates, GateResult{Gate: "smoke", Status: GatePass, Detail: "smoke PASS (" + firstNonEmpty(record.Smoke.EvidenceRef, "no evidence ref") + ")"})
	}
	return gates
}

// --- views ------------------------------------------------------------------------

func (service *Service) view(projectID string, state *State, release *Release) View {
	entry := *release
	view := View{Release: entry}
	if release.Status == StatusPromoted || release.Status == StatusRejected || release.Status == StatusPromotionFailed {
		view.LiveGates = release.Gates
		return view
	}
	view.LiveGates = service.regate(projectID, state, release)
	for _, gate := range view.LiveGates {
		if gate.Status == GateStale {
			view.Staleness = Staleness{Stale: true, Reason: gate.Detail}
			view.Release.Status = StatusStale
			if view.Release.Approval != nil && view.Release.Approval.Decision == "APPROVED" {
				approval := *view.Release.Approval
				approval.Decision = "STALE"
				view.Release.Approval = &approval
			}
			break
		}
	}
	return view
}

func (service *Service) receipt(state *State, release *Release, record DeploymentRecord) Receipt {
	refs := []string{}
	if release.Manifest.Build != nil && release.Manifest.Build.EvidenceRef != "" {
		refs = append(refs, release.Manifest.Build.EvidenceRef)
	}
	if record.Smoke != nil && record.Smoke.EvidenceRef != "" {
		refs = append(refs, record.Smoke.EvidenceRef)
	}
	build := BuildEvidence{}
	if release.Manifest.Build != nil {
		build = *release.Manifest.Build
	}
	approval := Approval{}
	if release.Approval != nil {
		approval = *release.Approval
	}
	receipt := Receipt{
		Kind: "ReleaseReceipt", SchemaVersion: "release-receipt/v1", ReceiptID: fmt.Sprintf("RR-%03d", state.NextReceipt),
		ReleaseID: release.ReleaseID, ProjectID: release.ProjectID, Status: record.Outcome, Transition: release.Manifest.Transition,
		Environment: release.Manifest.Environment, Changes: release.Manifest.Changes, Build: build, Approval: approval, Deployment: record,
		ManifestHash: release.Manifest.ManifestHash, EvidenceRefs: refs, IssuedAt: record.RecordedAt,
	}
	if promotion := release.Manifest.Promotion; promotion != nil {
		copied := *promotion
		receipt.Promotion = &copied
		receipt.EnvironmentDelta = append([]EnvironmentDeltaField{}, release.Manifest.EnvironmentDelta...)
		receipt.PreviousReceiptID = promotion.SourceReceiptID
		refs = append(refs, "receipt:"+promotion.SourceReceiptID)
		receipt.EvidenceRefs = refs
	}
	state.NextReceipt++
	receipt.ReceiptHash = hashJSON(receipt)
	return receipt
}

// --- helpers ------------------------------------------------------------------------

func environmentConfig(environment topology.Environment, version int) EnvironmentConfig {
	config := EnvironmentConfig{
		EnvironmentID: environment.ID, Type: environment.Type, TargetRef: environment.TargetRef,
		RequiredEvidence: append([]string{}, environment.RequiredEvidence...), ApproverPolicy: environment.ApproverPolicy, TopologyVersion: version,
	}
	// The hash covers what a deployment can drift on: target, evidence, approver policy.
	config.ConfigHash = hashJSON(map[string]any{"target_ref": config.TargetRef, "required_evidence": config.RequiredEvidence, "approver_policy": config.ApproverPolicy, "type": config.Type})
	return config
}

func manifestHash(manifest Manifest) string {
	copied := manifest
	copied.ManifestHash = ""
	return hashJSON(copied)
}

func verdictOf(gates []GateResult, approval *Approval, receipt *Receipt) (string, string) {
	if blocked := firstBlocked(gates); blocked != nil {
		return StatusGateBlocked, StatusGateBlocked
	}
	for _, gate := range gates {
		if gate.Gate == "build" && gate.Status == GateNeedsEvidence {
			return StatusAwaitingBuild, StatusAwaitingBuild
		}
	}
	if approval != nil && approval.Decision == "APPROVED" {
		return StatusApproved, StatusApproved
	}
	return StatusReadyForApproval, StatusReadyForApproval
}

func firstNotPassing(gates []GateResult) *GateResult {
	for index := range gates {
		if gates[index].Status != GatePass {
			return &gates[index]
		}
	}
	return nil
}

func firstBlocked(gates []GateResult) *GateResult {
	for index := range gates {
		if gates[index].Status == GateBlocked || gates[index].Status == GateStale {
			return &gates[index]
		}
	}
	return nil
}

func policyAllowsSingleOperator(environment EnvironmentConfig) bool {
	for _, evidence := range environment.RequiredEvidence {
		if strings.EqualFold(evidence, "single-operator-controls") {
			return true
		}
	}
	return false
}

func (service *Service) load(projectID string) (*State, error) {
	if _, err := service.topology.Get(projectID); err != nil {
		var typed *topology.Error
		if asTopologyError(err, &typed) && typed.Code == topology.CodeProjectNotFound {
			return nil, newError(CodeProjectNotFound, "project %q is not configured", projectID)
		}
		return nil, newError(CodeStateUnavailable, "topology unavailable: %v", err)
	}
	state, err := service.store.Load(projectID)
	if err != nil {
		return nil, err
	}
	if state == nil {
		state = &State{Kind: Kind, SchemaVersion: SchemaVersion, ProjectID: projectID, NextRelease: 1, NextReceipt: 1, Releases: []Release{}, Audit: []AuditEvent{}}
	}
	return state, nil
}

func asTopologyError(err error, target **topology.Error) bool {
	typed, ok := err.(*topology.Error)
	if ok {
		*target = typed
	}
	return ok
}

func (service *Service) stamp() string { return service.now().Format(time.RFC3339) }

func (service *Service) audit(state *State, at, actor, action, object, reason string) {
	state.Audit = append(state.Audit, AuditEvent{Sequence: len(state.Audit) + 1, At: at, Actor: actor, Action: action, Object: object, Reason: reason})
}

func findRelease(state *State, releaseID string) *Release {
	for index := range state.Releases {
		if state.Releases[index].ReleaseID == releaseID {
			return &state.Releases[index]
		}
	}
	return nil
}

func uniqueTrimmed(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" || seen[trimmed] {
			continue
		}
		seen[trimmed] = true
		out = append(out, trimmed)
	}
	return out
}

func shortHash(value string) string {
	if len(value) > 19 {
		return value[:19]
	}
	return value
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
