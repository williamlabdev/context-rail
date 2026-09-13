# ContextRail Architecture

Status: proposed implementation architecture
Version: 1.2
Updated: 2026-09-13
Implementation constraint: React frontend, Go backend, Cloud Run-first prototype

## 1. Architecture in one sentence

ContextRail is a Go-based decision control plane that reconciles private source and cloud evidence into a versioned Project graph, uses RAG to explain the evidence, applies deterministic policy and cost rules, and requires human gates before an AI-assisted change advances through a Project's custom environment topology.

### Prototype market boundary

The P0 architecture is optimized for startup and SMB engineering teams of roughly 5–30 people: teams with Git/CI and an AI coding assistant, at least two delivery environments, and no dedicated platform team. The first application may be an internal system; B2B SaaS is the next validation cohort. This is a prototype adoption boundary, not a hard technical restriction.

The model keeps explicit extension points for larger enterprises, but does not claim to implement them in P0. Enterprise SSO/SCIM, fine-grained ACL, private-network/on-prem collectors, organization-wide inventory, multiple provider reconciliation, multi-cloud and production operations remain later capabilities whose value must be validated after the startup/SMB workflow is repeatable.

## 2. Terms and boundaries

### Project is not a repository

Project is the smallest ContextRail governance boundary. It owns the business/system identity, people, documents, repositories, services, environments, policies, changes and evidence lineage required to decide whether a change can move forward.

Repository is a source-control entity discovered or imported from GitHub, GitLab, self-hosted GitLab, a manifest or a fixture. A Project may connect to many repositories. A repository may be linked to many Projects, but every link must state why.

GCP Project is an infrastructure/account boundary. A ContextRail Project can deploy to different GCP Projects for different environments; one GCP Project may host several application Projects. The relationship is through an environment deployment target, not a name match.

### Project document boundary

The Project document set is a governed source boundary, not a generic upload folder. P0 uses the [Project Context Contract](docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md) to declare required documents, audience, authority role, source version, effective time, access scope and status. The minimum set is `project.yaml`, `README.md`, `vision.md`, `architecture.md`, `docs/engineering/development.md`, `docs/operations/environments.md` and `AGENTS.md`; policies, ADRs, glossaries and service schemas are additional sources when the Project has them.

`ProjectDocument` is the source registry. `ContextPack` is a derived, rebuildable projection for AI retrieval and agent handoff. It must retain source paths, version／commit, content hashes and generation metadata, and it must never become the SSOT or fill missing information. A Project can be decision-ready while still missing development-only documents, but it cannot be described as development- or staging-ready until the corresponding document and evidence gates pass.

### Relationship policy

The storage model supports many-to-many relationships, while governance rules prevent ambiguous ownership:

| Relationship | Cardinality | Required rule |
| --- | --- | --- |
| ContextRail Project ↔ Repository | M:N via ProjectRepository | At most one active primary owner per repository; other links are shared dependency, infrastructure or path-scoped links. |
| ContextRail Project ↔ Service | M:N via ProjectService | A shared runtime must be explicitly marked as a dependency; a change still has one primary Project. |
| ContextRail Project ↔ Environment | 1:N via ProjectEnvironment | Environment identity is stable; topology versions are immutable snapshots. |
| Environment ↔ GCP deployment target | M:N via EnvironmentTarget | Target includes provider instance, GCP project, region, service and configuration evidence. |
| ContextRail Project ↔ Document source | M:N via ProjectSource | Each link records access scope, source version and confirmation state. |
| ContextRail Project ↔ ContextRail Project | M:N via ProjectDependency | Dependency type and direction are explicit; no hidden cross-project impact. |
| ContextRail Project ↔ Change | 1:N | Every Change/Ticket has exactly one primary project_id; it may reference dependent Projects. |
| ContextRail Project ↔ Release Bundle | 1:N | A bundle groups one release attempt's Changes while preserving each Change's own decision and evidence. |
| Release Bundle ↔ Change | M:N via ReleaseBundleChange | A Change may ship independently or in a bundle; bundling cannot bypass its required gate. |

## 3. System context

~~~text
┌──────────────────────────────────────────────────────────────┐
│                         ContextRail                         │
│  Project Registry · Workspace · Decision · Evidence · Gate   │
└───────────────┬─────────────────────┬────────────────────────┘
                │                     │
      read/normalize evidence       human decisions
                │                     │
┌───────────────▼──────────────┐   ┌──▼─────────────────────────┐
│ Sources and execution systems │   │ Existing engineering tools │
│ GitHub / GitLab / manifests   │   │ Git PR · CI · Jira/Issues  │
│ Cloud Build / Cloud Run       │   │ Claude Code · Codex         │
│ Artifact Registry / documents │   │ existing review controls    │
└──────────────────────────────┘   └────────────────────────────┘
~~~

ContextRail does not replace Git, Jira, CI runners, Cloud Deploy, Terraform, coding agents or inline code-review tools. It consumes their evidence and controls the decision boundary between accepted intent, candidate change and governed promotion.

## 4. Logical architecture

~~~text
┌───────────────────────────────────────────────────────────────┐
│ React Web UI                                                   │
│ Project Registry · Project Workspace · Documents/AI Context · Reports │
└───────────────────────────────┬───────────────────────────────┘
                                │ HTTPS / identity
┌───────────────────────────────▼───────────────────────────────┐
│ Go API / modular monolith                                     │
│ project · source · change · decision · agent · evidence        │
│ environment · architecture · cost · release · audit           │
└─────────────┬───────────────┬──────────────┬─────────────────┘
              │               │              │
       domain contracts   deterministic   provider adapters
              │            decision rules   and collectors
┌─────────────▼───────┐ ┌───▼────────────┐ ┌─▼─────────────────┐
│ Firestore            │ │ RAG / Gemini   │ │ Cloud/Git sources │
│ records + relations  │ │ retrieval +    │ │ metadata + diff   │
│ versions + audit     │ │ explanation    │ │ build/runtime     │
└─────────────┬───────┘ └───┬────────────┘ └───────────────────┘
              │              │
       ┌──────▼──────┐ ┌────▼──────────┐
       │ Cloud Storage│ │ Cloud Build   │
       │ evidence     │ │ image/digest  │
       │ and reports  │ │ and deploy    │
       └──────────────┘ └────┬──────────┘
                             │
                       Cloud Run targets
                       staging / prod-demo
~~~

The first implementation is a modular monolith rather than a fleet of microservices. Boundaries are enforced through packages and contracts so that the five-week prototype does not spend its budget on distributed-systems plumbing.

## 5. Domain model

### Core entities

| Entity | Purpose | Important fields or invariants |
| --- | --- | --- |
| TenantContext | Tenant-ready partitioning without claiming commercial multi-tenancy | tenant_id, actor, membership, access scope |
| Project | Business/system governance boundary | project_id, key, owner, business owner, status, version, classification, config hash |
| ProjectDocument | Declared source document and readiness state | project_id, path, kind, audience, authority, source version, effective time, access scope, status, index state, evidence refs |
| ContextPack | Rebuildable AI-facing projection of confirmed sources | pack_id, project_id, schema version, source document IDs, source hash, generated_at, status, missing/conflict list; never SSOT |
| Repository | Canonical source identity | provider, instance URL, provider repo ID, URL, default branch, last observed revision |
| ProjectRepository | Project-to-repo relationship | role, path, primary flag, ownership, confirmation, evidence refs |
| Service | Logical or deployable application service | name, runtime, source path, container/image identity |
| ProjectService | Project-to-service relationship | primary/dependency role, service path, ownership |
| Environment | Project-defined promotion node | stable ID, display name, standard type, order, status |
| EnvironmentTopology | Immutable ordered environment snapshot | version, predecessor/successor, target references, config hash |
| PromotionPolicy | Allowed transition and required evidence | action, predecessor, successor, approver separation, expiry |
| Change | Feature, requirement change, rebuild or bug fix | one primary project_id, source request, affected scope, state |
| DecisionRecord | Single source of truth for an accepted or rejected decision | input versions, selected option, cost assumptions, unknowns, decision ID |
| AgentWorkOrder | Bounded execution contract | repo/path/branch, allowed paths, forbidden actions, acceptance IDs, hash, expiry |
| AgentRunRecord | Agent execution declaration and evidence | agent/provider/model/version, work-order hash, run ID, commit, provenance limits |
| Candidate | Proposed implementation state | PR, diff, test and review references, candidate decision |
| EvidenceBundle | Structured engineering proof | Git, review, CI, build, environment and runtime evidence |
| ReleaseBundle | Release-level grouping of one or more Changes | bundle_id, project_id, change IDs, release intent, target transition, approval state |
| ReleaseManifest | Release identity | commit, image digest, target, config hash, policy version |
| ReleaseReceipt | Re-readable result of a governed release | approval, operation, revision, smoke result, evidence refs |
| AuditEvent | Append-only governance history | actor, action, object, version, reason, timestamp |

### Environment lifecycle

Environment records have a lifecycle independent from the Project lifecycle:

~~~text
DRAFT → ACTIVE → PAUSED → RETIRED
                 ↑          │
                 └──────────┘  only through a new topology version
~~~

The UI and API must provide:

- Add: create a new environment in a new EnvironmentTopology version.
- Edit: update name, standard type, target, order, owner or policy by creating a new version.
- Retire: stop selecting the environment for new Changes while retaining historical evidence.
- Restore: reintroduce it through a validated new topology version.
- No hard delete for referenced environments.

Not every edit invalidates the same records. A display-only label change can remain informational; target, standard type, order, promotion rule, approver, required evidence or configuration changes invalidate dependent decisions and approvals. The invalidation reason must be visible.

Suggested API semantics:

~~~text
GET   /v1/projects/{project_id}/environments
POST  /v1/projects/{project_id}/environments
PATCH /v1/projects/{project_id}/environments/{environment_id}
POST  /v1/projects/{project_id}/environments/{environment_id}:retire
POST  /v1/projects/{project_id}/environments/{environment_id}:restore
~~~

There is deliberately no P0 hard-delete endpoint. If a never-used draft must be discarded, it is an unreferenced draft cleanup operation, not deletion of an active evidence-bearing environment.

## 6. Private repository discovery and source coverage

### What the system can know

ContextRail can observe only the sources in its authorized scope. A source connection has:

~~~text
SourceConnection
  provider: github | gitlab | self_hosted_gitlab | manifest | fixture
  instance_url
  organization_or_group
  credential_ref
  requested_scope: metadata | read_code | read_ci
  last_sync_at
~~~

Private credentials are references to Secret Manager or an equivalent runtime secret store. The API never returns token values to the browser. Default provider operations are read-only.

Repository identity uses provider + instance_url + provider_repo_id, not only a repository name or URL. This prevents a rename or fork from silently creating a duplicate source.

### Discovery is not Project assignment

~~~text
Source connection
      ↓
Repository inventory
      ↓
Candidate ProjectRepository links
      ↓
Human confirmation or manifest confirmation
      ↓
Project baseline
~~~

Signals for candidate links include repository metadata, handoffguard.yaml, CODEOWNERS, build triggers, Cloud Run labels/provenance, deployment configuration, source paths, service names and authorized documents. These signals produce INFERRED candidates with evidence references; they do not silently create Project ownership.

Coverage statuses are explicit:

~~~text
DISCOVERED → LINKED → CONFIRMED → STALE
                    └→ NEEDS_INPUT
UNLINKED_REPOSITORY · UNKNOWN_DEPLOYMENT_SOURCE · ORPHANED_SERVICE
~~~

RAG can read an architecture document that mentions a repository, but only inventory reconciliation can say that an observed repository is not yet linked. Absence outside the connected observation scope remains UNKNOWN.

## 7. RAG, evidence graph and decision engine

The three layers have different responsibilities:

| Layer | Question it answers | Must not claim |
| --- | --- | --- |
| Evidence/Inventory | What source or runtime fact was observed, when and with what permission? | Complete enterprise inventory if the source was not connected. |
| Evidence Graph | Which Project, Repo, path, Service, Environment, Change and Receipt are related? | That a semantic relationship is true without confirmation/evidence. |
| RAG | Which versioned documents explain the observed system and what information is missing? | That a missing retrieval result proves a missing document or Repo. |
| Decision Engine | Given policy, evidence and human inputs, is the next action PASS, FAIL, NEEDS_REVIEW or UNKNOWN? | Unilaterally approving, changing IAM, inventing cost or promoting to production. |

P0 can implement the graph as Firestore collections and relation documents. A separate graph database is not required. Embeddings are retrieval indexes, not the system of record. Confirmed decisions and evidence are retained separately from model output so future RAG improvements cannot rewrite history.

The document registry is evaluated before retrieval: required documents are classified as `CURRENT`, `DRAFT`, `MISSING`, `STALE`, `CONFLICT` or `REVOKED`; a context pack may be `CURRENT` or `PARTIAL`, but the pack status cannot upgrade a source document. Validation and rebuild are explicit actions with an audit event. A retrieval miss means “not retrieved in the authorized scope”, not proof that the organization has no document or repository.

## 8. Decision and release flow

~~~text
Project / source baseline
        ↓
Change request
        ↓
RAG retrieval + architecture/cost candidates
        ↓
Human accepts or requests input
        ↓
DecisionRecord (immutable version)
        ↓
AgentWorkOrder (bounded, hashed, expiring)
        ↓
Agent run → feature branch → PR → independent review
        ↓
Git/CI/Build evidence reconciliation
        ↓
Candidate decision
        ↓
ReleaseBundle (one or more Changes)
        ↓
Cloud Run staging verification
        ↓
Release approval + promotion policy
        ↓
Cloud Run prod-demo + smoke test
        ↓
ReleaseReceipt
~~~

The same decision_id, source snapshot hash, topology version, policy version and evidence references connect the human Change Decision Brief to the Agent Context Pack. A required input or policy change makes the dependent work order, candidate or approval stale; it does not silently regenerate a new approval.

### Agent communication boundary

P0 treats agent roles as bounded producers and verifiers, not autonomous conversational peers. A Context/Architecture Agent may produce an impact assessment; an Engineering Agent executes an accepted `AgentWorkOrder`; an Evidence Agent reconciles implementation evidence; and a Release Policy Agent evaluates deterministic rules. None of these roles may grant IAM, approve production, bypass a human gate or convert an unknown state into pass.

The P0 handoff is artifact-to-artifact and must carry at least `project_id`, `request_id`, `decision_id`, `source_snapshot_hash`, `policy_version`, `input_refs`, `output_artifact`, `evidence_refs`, `status` and `human_gate`. Free-form conversation is not evidence, and model output does not become an accepted project fact without a canonical record and the required human confirmation.

P0 does not require or implement autonomous Agent-to-Agent orchestration. Internal handoffs use modular-monolith contracts and persisted artifacts. A2A remains a conditional P1 extension only when independent remote agents, providers or runtimes create a demonstrated interoperability need. Even then, A2A is limited to task/artifact interoperability; `DecisionRecord`, `EvidenceBundle`, authorization and release policy remain ContextRail-owned boundaries.

### Release Bundle semantics

The unit of analysis is a Change; the unit of promotion may be a Release Bundle. A bundle can contain a feature, a requirement change and/or a bug fix from the same Project release window. Each Change keeps its own scope, DecisionRecord, work order, candidate and evidence status. The release gate evaluates the complete bundle, reports which Change is blocking, and never treats a green sibling Change as evidence for another one. The P0 demo uses 2–3 bundled Changes to reflect normal small-team releases without building a full release-train or dependency-planning product.

## 9. Google Cloud deployment shape

### Product runtime

- Cloud Run: ContextRail API and static React delivery.
- Firestore: Project records, relation documents, versions, decisions, policies and audit events.
- Cloud Storage: evidence bundles, rendered reports and receipt artifacts with controlled retention.
- Gemini/Vertex AI: structured extraction, retrieval-assisted explanation and candidate generation.
- Embeddings/vector retrieval: versioned source/chunk index; not the policy source of truth.
- Secret Manager: provider credentials and runtime secrets by reference.
- Cloud Build and Artifact Registry: product build and demo candidate build evidence.
- Firebase Authentication or an equivalent identity layer for the prototype; membership checks remain server-side.

### Target application runtime

The demo application has separate Cloud Run staging and prod-demo services. A promotion carries the same approved image digest but creates a new target revision and independently verifies target configuration. Digest equality alone is not a release proof.

The prototype can define Development, Testing, Staging, Production, UAT or other user-named environments, but it executes real deployment only for staging and isolated prod-demo. Production is represented as protected/read-only unless the competition scope is explicitly changed.

## 10. Go package boundary

~~~text
cmd/handoffguard
internal/api
internal/domain/project       internal/domain/repository
internal/domain/environment   internal/domain/change
internal/domain/decision      internal/source
internal/rag                  internal/graph
internal/architecture         internal/cost
internal/agent                internal/evidence
internal/release              internal/persistence
contracts · pricing · infra · tests
~~~

Keep the first version as a modular monolith. Each package should expose a small contract and preserve the provenance of inputs. Do not create a microservice for each AI step.

## 11. Security and failure semantics

- Read-only source access is the default; write access is isolated to explicit release operations.
- Provider tokens are never sent to the React client or placed in an Agent Work Order.
- Agent instructions cannot grant IAM, approve a Change or deploy to production.
- Untrusted text from documents, PRs and issue descriptions is data, not executable policy.
- Every decision records PASS, FAIL, NEEDS_REVIEW or UNKNOWN; unknown is not coerced to pass.
- Source revocation, document version change, topology/policy change and target drift invalidate only dependent artifacts and state why.
- Operation IDs and idempotency keys prevent duplicate promotion; a timeout is reconciled against the actual cloud operation before retry.
- Archive is recoverable from the evidence perspective; hard deletion is outside P0.
- Complete private-network data residency, fine-grained document ACL and cross-resource rollback remain productisation work, not prototype claims.

## 12. P0 / P1 / P2 boundary

| Layer | Included | Excluded |
| --- | --- | --- |
| P0 | One Cloud Run application for a startup/SMB-first demo, Project Registry, M:N repository links, custom environment topology, versioned policy, ProjectDocument registry, manifest validation, derived Context Pack with source lineage, RAG citations, deterministic cost, Agent Work Order, structured agent handoff, one external agent run, PR/review/CI evidence, a 2–3 Change Release Bundle, staging, prod-demo and receipt. | Enterprise-wide discovery, arbitrary agent hosting, generic document ingestion, full production readiness, hard delete, autonomous FDE, full rollback, autonomous A2A. |
| Conditional P1 | At most one narrow adapter after G4: provider inventory, fixed agent, FDE handoff pack, Drive/Docs, ADC REST or revision rollback. | Combining all adapters into the prototype. |
| P2 | Multi-provider reconciliation, private-network collector, fine-grained ACL, multi-cloud, Cloud Assist MCP, complete rollback, autonomous agent orchestration and production operations. | No implied completion date in the five-week competition build. |

## 13. Architecture decisions still requiring confirmation

1. Which provider is the first real source: GitHub, GitLab Cloud, self-hosted GitLab or manifest-only?
2. Is the demo a monorepo, one repository or a polyrepo application?
3. Which service/path is the primary owner when a repository is shared?
4. What is the first environment transition matrix and which approvals are mandatory?
5. Which GCP identity can read source/build/runtime evidence and deploy only to staging/prod-demo?
6. Which document types are allowed in the prototype and what is their retention period?
7. Which narrow P1, if any, is more valuable than finishing negative tests and reproduction evidence?
8. Which startup/SMB design-partner case will be used first, and what later enterprise extension signal would justify expanding the boundary?

Until these are answered, do not collapse Project, Repository, GCP Project, Service or Environment into one identifier. That shortcut would make cross-Project impact analysis and stale approval detection unreliable.

## 14. UI is a projection of governed state

The P0 UI must follow the [UI Flow & State Contract](docs/design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md). It is not an independent source of truth and it must not imply that a toast, static fixture or report preview changed governance state.

Every mutating action is valid only when its preconditions pass. After a successful mutation, the UI re-reads the canonical Project, Change, DecisionRecord, Gate or Receipt instead of trusting local browser state. The response must expose the actor, version, before／after status, audit event and evidence references. A failed or preview action keeps the prior state and is labelled accordingly.

The first implementation gate is the single UI path `Project → Documents/AI Context → Change → Decision → Work Order → Evidence → staging → Receipt`, including one missing-document and one `NEEDS_INPUT` block. Separate screens for Changes, Impact, Evidence, Reports and Policy Gate must either contain the contracted content or be explicitly disabled; navigation that only changes a toast is not a completed workflow. The same canonical state is presented through a `zh-TW`／`en` locale switcher: human labels are localized, while project names, IDs, paths, commits, digests and other technical identifiers remain unchanged.

## 15. Related documents

- [Vision](vision.md)
- [Roadmap](roadmap.md)
- [Project Plan v4.2](docs/planning/CONTEXT_RAIL_PROJECT_PLAN_V4.2.zh-TW.md)
- [Critical Thinking Review](docs/reviews/CONTEXT_RAIL_CRITICAL_REVIEW.zh-TW.md)
- [UI Flow & State Contract](docs/design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md)
- [Project Context Contract](docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md)
