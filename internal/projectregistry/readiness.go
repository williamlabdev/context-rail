package projectregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var acceptedDecisionStatuses = map[string]bool{
	"ACCEPTED":                 true,
	"ACCEPTED_FOR_DEVELOPMENT": true,
	"ACCEPTED_FOR_STAGING":     true,
}

func readinessFor(root string, spec manifestSpec, documents []DocumentStatus, decisions []DecisionSummary, rawDecisions []map[string]any) (ReadinessSet, error) {
	documentByPath := make(map[string]DocumentStatus, len(documents))
	for _, document := range documents {
		documentByPath[document.Path] = document
	}
	decisionBlockers := []string{}
	for _, document := range spec.Documents {
		if contains(document.RequiredFor, "decision") {
			if observed, ok := documentByPath[document.Path]; ok {
				if contains([]string{"MISSING", "STALE", "CONFLICT", "NEEDS_INPUT"}, observed.Status) {
					decisionBlockers = append(decisionBlockers, observed.Status)
				}
			}
		}
	}
	decisionReady := ReadinessStatus{Status: "READY", Reasons: []string{}}
	if len(decisionBlockers) > 0 {
		decisionReady = ReadinessStatus{Status: "NEEDS_INPUT", Reasons: []string{"required decision documents are not all CURRENT"}}
	}

	accepted := []map[string]any{}
	hasAcceptedStatus := false
	for _, decision := range rawDecisions {
		status, _ := decision["status"].(string)
		if acceptedDecisionStatuses[status] {
			hasAcceptedStatus = true
		}
		if humanDecisionAccepted(decision, "accept_request") {
			accepted = append(accepted, decision)
		}
	}
	localReasons := []string{}
	if len(accepted) == 0 {
		localReasons = append(localReasons, "no human-accepted DecisionRecord found")
	}
	if hasAcceptedStatus && len(accepted) == 0 {
		localReasons = append(localReasons, "DecisionRecord status conflicts with pending human decision")
	}
	for _, document := range spec.Documents {
		if contains(document.RequiredFor, "development") {
			if observed, ok := documentByPath[document.Path]; !ok || observed.Status != "CURRENT" {
				localReasons = append(localReasons, "required development documents are not all CURRENT")
				break
			}
		}
	}
	if len(accepted) > 0 {
		matches := false
		for _, decision := range accepted {
			if decisionMatchesContextSnapshot(decision, documentByPath, root) {
				matches = true
				break
			}
		}
		if !matches {
			localReasons = append(localReasons, "accepted DecisionRecord does not match the current Context Pack snapshot")
		}
	}
	local := ReadinessStatus{Status: "READY", Reasons: []string{}}
	if len(localReasons) > 0 {
		local = ReadinessStatus{Status: "NEEDS_INPUT", Reasons: localReasons}
	}

	cloudEvidence, cloudReasons := cloudTestingEvidence(root)
	if local.Status != "READY" {
		cloudReasons = append([]string{"local development readiness is not READY"}, cloudReasons...)
	}
	cloud := ReadinessStatus{Status: "READY", Reasons: []string{}}
	if cloudEvidence != "READY" || len(cloudReasons) > 0 {
		cloud = ReadinessStatus{Status: "NEEDS_INPUT", Reasons: cloudReasons}
	}

	stagingReasons := []string{}
	stagingEnvironment := map[string]any{}
	for _, environment := range spec.Environments {
		if environment.ID == "staging" {
			stagingEnvironment = map[string]any{"targetRef": environment.TargetRef}
			break
		}
	}
	if target, _ := stagingEnvironment["targetRef"].(string); target == "" || strings.Contains(target, "<REPLACE_ME") {
		stagingReasons = append(stagingReasons, "staging target is not configured")
	}
	if cloud.Status != "READY" {
		stagingReasons = append(stagingReasons, "cloud testing readiness is not READY")
	}
	stagingAccepted := false
	for _, decision := range rawDecisions {
		if humanDecisionAccepted(decision, "allow_staging") {
			stagingAccepted = true
			break
		}
	}
	if !stagingAccepted {
		stagingReasons = append(stagingReasons, "no human-approved staging DecisionRecord found")
	}
	staging := ReadinessStatus{Status: "READY", Reasons: []string{}}
	if len(stagingReasons) > 0 {
		staging = ReadinessStatus{Status: "NEEDS_INPUT", Reasons: stagingReasons}
	}

	stagingEvidence, stagingEvidenceReasons := evidence(root, stagingTarget(spec))
	verifiedReasons := append([]string(nil), stagingEvidenceReasons...)
	if staging.Status != "READY" {
		verifiedReasons = append([]string{"staging deployment readiness is not READY"}, verifiedReasons...)
	}
	verified := ReadinessStatus{Status: "READY", Reasons: []string{}}
	if stagingEvidence != "READY" || len(verifiedReasons) > 0 {
		verified = ReadinessStatus{Status: "NEEDS_INPUT", Reasons: verifiedReasons}
	}

	return ReadinessSet{
		ReadyForDecision:         decisionReady,
		ReadyForLocalDevelopment: local,
		ReadyForCloudTesting:     cloud,
		ReadyForStaging:          staging,
		StagingVerified:          verified,
		Production:               ReadinessStatus{Status: "BLOCKED", Reasons: []string{"production is human-gated and read-only in P0"}},
	}, nil
}

func contains(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func humanDecisionAccepted(decision map[string]any, action string) bool {
	status, _ := decision["status"].(string)
	if !acceptedDecisionStatuses[status] {
		return false
	}
	for _, field := range []string{"human_decision", "human_decisions"} {
		value, ok := decision[field].(map[string]any)
		if !ok {
			continue
		}
		actionValue, ok := value[action]
		if !ok {
			continue
		}
		if actionMap, ok := actionValue.(map[string]any); ok {
			actionValue = actionMap["decision"]
		}
		if strings.EqualFold(fmt.Sprint(actionValue), "ACCEPTED") || strings.EqualFold(fmt.Sprint(actionValue), "APPROVED") || strings.EqualFold(fmt.Sprint(actionValue), "TRUE") {
			return true
		}
	}
	return false
}

func decisionMatchesContextSnapshot(decision map[string]any, documents map[string]DocumentStatus, root string) bool {
	expected, _ := decision["source_snapshot_hash"].(string)
	contextDocument, ok := documents["docs/ai/context-pack.json"]
	if expected == "" || !ok || contextDocument.Status != "DERIVED" {
		return false
	}
	path, err := safeDocumentPath(root, contextDocument.Path)
	if err != nil {
		return false
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	var pack struct {
		SourceSnapshotHash string `json:"source_snapshot_hash"`
	}
	if json.Unmarshal(data, &pack) != nil {
		return false
	}
	return pack.SourceSnapshotHash == expected
}

func cloudTestingEvidence(root string) (string, []string) {
	evidenceRoot := filepath.Join(root, "evidence")
	if info, err := os.Stat(evidenceRoot); err != nil || !info.IsDir() {
		return "NEEDS_INPUT", []string{"Evidence Bundle is missing"}
	}
	reasons := []string{}
	testFiles := matchingFiles(evidenceRoot, "test-output.txt")
	if len(testFiles) == 0 || anyFileMissingPass(testFiles) {
		reasons = append(reasons, "test evidence is not PASS")
	}
	buildFiles := matchingFiles(evidenceRoot, "build-output.txt")
	if len(buildFiles) == 0 || anyFileMissingPass(buildFiles) {
		reasons = append(reasons, "build evidence is not PASS")
	}
	status, reviewReasons := reviewEvidence(root)
	if status != "READY" {
		reasons = append(reasons, reviewReasons...)
	}
	if len(reasons) == 0 {
		return "READY", reasons
	}
	return "NEEDS_INPUT", reasons
}

func evidence(root, target string) (string, []string) {
	evidenceRoot := filepath.Join(root, "evidence")
	if info, err := os.Stat(evidenceRoot); err != nil || !info.IsDir() {
		return "NEEDS_INPUT", []string{"Evidence Bundle is missing"}
	}
	reasons := []string{}
	status, reviewReasons := reviewEvidence(root)
	if status != "READY" {
		reasons = append(reasons, reviewReasons...)
	}
	receipts := matchingFiles(evidenceRoot, "receipt.json")
	if len(receipts) == 0 {
		reasons = append(reasons, "release receipt is missing")
	} else {
		matchingPass := false
		for _, path := range receipts {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			var receipt map[string]any
			if json.Unmarshal(data, &receipt) != nil {
				continue
			}
			if receipt["status"] == "PASS" && receipt["environment"] == "staging" && receipt["target"] == target {
				matchingPass = true
			}
		}
		if !matchingPass {
			reasons = append(reasons, "no PASS staging receipt matches the manifest target")
		}
	}
	if len(reasons) == 0 {
		return "READY", reasons
	}
	return "NEEDS_INPUT", reasons
}

func reviewEvidence(root string) (string, []string) {
	paths := matchingFiles(filepath.Join(root, "evidence"), "code-review.md")
	if len(paths) == 0 {
		return "NEEDS_INPUT", []string{"independent code review is missing"}
	}
	reasons := []string{}
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			reasons = append(reasons, "code review cannot be read: "+path)
			continue
		}
		text := string(data)
		status := strings.ToUpper(markdownField(text, "Status"))
		reviewer := strings.ToLower(markdownField(text, "Reviewer"))
		if status != "PASS" {
			reasons = append(reasons, fmt.Sprintf("independent code review is %s: %s", fallback(status, "UNKNOWN"), path))
		}
		if reviewer == "" || reviewer == "UNASSIGNED" || reviewer == "UNKNOWN" || reviewer == "PENDING" {
			reasons = append(reasons, "code review has no independent reviewer identity: "+path)
		}
		if status == "PASS" && markdownField(text, "Reviewer actor_id") == "" {
			reasons = append(reasons, "code review has no reviewer actor_id: "+path)
		}
		if status == "PASS" && markdownField(text, "Reviewed commit") == "" {
			reasons = append(reasons, "code review has no reviewed commit: "+path)
		}
	}
	if len(reasons) == 0 {
		return "READY", reasons
	}
	return "NEEDS_INPUT", reasons
}

func markdownField(text, label string) string {
	prefix := label + ":"
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, prefix) {
			return strings.Trim(strings.TrimSpace(strings.SplitN(line, ":", 2)[1]), "`")
		}
	}
	return ""
}

func fallback(value, fallbackValue string) string {
	if value == "" {
		return fallbackValue
	}
	return value
}

func matchingFiles(root, suffix string) []string {
	var paths []string
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err == nil && info != nil && !info.IsDir() && strings.HasSuffix(path, suffix) {
			paths = append(paths, path)
		}
		return nil
	})
	return paths
}

func anyFileMissingPass(paths []string) bool {
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), "Result: PASS") {
			return true
		}
	}
	return false
}

func stagingTarget(spec manifestSpec) string {
	for _, environment := range spec.Environments {
		if environment.ID == "staging" {
			return environment.TargetRef
		}
	}
	return ""
}
