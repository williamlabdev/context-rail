# ContextRail Roadmap

Status: working product and engineering baseline
Version: 1.2
Updated: 2026-09-13
Team: 2 people, 5-week prototype window (dates remain provisional until G0 confirms the official submission deadline)

## 1. Roadmap decision

ContextRail is an environment-aware AI Change Assurance layer for cloud-native business systems. Its first delivery slice is designed for startup and SMB engineering teams, using Cloud Run, Gemini, Git/CI evidence and a controlled staging-to-prod-demo path.

The core unit is a **Project governance boundary**, not a Git repository and not a Google Cloud Project. A Project may contain many repositories, services, documents and environments. A repository may be connected to more than one Project through an explicit, role-bearing relationship; the system must not silently treat 'one repo = one project'.

The product has one primary job:

> Before an AI-assisted change advances to the next governed environment, prove that the change still matches the accepted intent, scope, policy and release identity.

RAG, repository discovery, architecture advice, cost analysis, agent handoff and deployment evidence are supporting stages in this job. They are not separate products.

### Agent collaboration boundary

P0 uses structured Agent Handoff instead of autonomous Agent-to-Agent conversation. Context/Architecture, Engineering, Evidence and Release Policy roles exchange versioned artifacts; human acceptance and deterministic gates remain authoritative. A2A is a conditional P1 option only if independent remote agents, providers or runtimes create a measured interoperability need. It is not a prerequisite for the core path.

### Prototype market decision

P0 is explicitly **startup/SMB-first**, not enterprise-first. The initial design partner is a 5–30 person engineering team that already uses Git PRs, CI and an AI coding assistant, manages at least two delivery environments, and does not have a dedicated platform team. The first use cases are internal systems and then B2B SaaS; a company does not need to be classified by revenue or employee count if it has these operating characteristics.

Large enterprises remain an **enterprise-ready extension space**, not a P0 buyer-validation requirement. Enterprise SSO/SCIM, fine-grained ACL, private-network connectors, organization-wide inventory, multi-provider reconciliation, multi-cloud and production operations are intentionally reserved for later validation. P0 measures setup time, repeat use, decision quality and willingness to continue with startup/SMB teams before expanding the product boundary.

## 1.1 Project Context Contract

Every Project needs a small, explicit document baseline before its AI context can be considered usable. P0 standardizes the minimum contract as `project.yaml`, `README.md`, `vision.md`, `architecture.md`, `docs/engineering/development.md`, `docs/operations/environments.md` and `AGENTS.md`. Each document has a declared audience, authority role, source version, effective time, access scope and status. The contract is a ContextRail product proposal, not a claim that the industry has one universal AI-project template.

`ProjectDocument` is the registry of source documents. `docs/ai/context-pack.json` is a derived, rebuildable view that must retain path, version／commit, hash, effective time and access scope for every source. RAG indexes and context packs never become SSOT, never silently replace an accepted DecisionRecord, and never turn missing information into a fact. The Project Workspace must show missing, stale, conflicting, revoked and derived states together with the readiness reason.

The complete directory suggestion, import/validation/rebuild semantics and decision/development/staging readiness gates are defined in [Project Context Contract](docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md). The first UI slice is a Documents / AI Context page; it is a readiness and provenance surface, not a generic file-upload product.

## 2. Environment management decision

The earlier UI showing only '＋ 新增環境' was incomplete, not an intentional product limitation.

P0 supports environment topology management with the following semantics:

| Operation | P0 decision | Governance behaviour |
| --- | --- | --- |
| Add | Supported | Add an environment to a new immutable topology version; validate its standard type, order, target, owner and required evidence. |
| Edit | Supported | Update display metadata, target, order or policy through a new topology version and audit event. Changes to target, transition, approver or required evidence mark affected decisions and approvals 'STALE' or 'NEEDS_REVIEW'. |
| Reorder | Supported | Rebuild the topology version and revalidate predecessor/successor transitions. |
| Disable / remove | Supported as retire | Use 'RETIRED', not physical deletion. Existing receipts and evidence remain readable; new changes cannot select the retired environment. |
| Restore | Controlled | Reactivate only through a new topology version after validating target and promotion policy. |
| Hard delete | Not supported | No hard delete for an environment or any record referenced by a decision, evidence bundle or release receipt. |

'Production' may be renamed for display, but its standard type and minimum protection cannot be bypassed. ContextRail manages the Project's policy and evidence contract; it does not become the source of truth for Cloud Run, Terraform or IAM resources.

## 3. Five-week P0 roadmap

P0 is one complete vertical slice, not a collection of disconnected demos. Before Go API implementation begins, the UI Flow & State Contract must define and validate the user-visible state transitions, action preconditions, role boundaries, persistence and evidence for that slice. A static screen or toast is not an accepted implementation substitute.

| Week | Focus | Exit condition |
| --- | --- | --- |
| Week 0 / G0 | Confirm official deadline, available GCP project, billing, quotas, Go toolchain, model access, Git source access and the exact demo application; freeze the UI Flow & State Contract and run its first interaction walkthrough. | A written scope freeze, working cloud-access checklist, and a reviewed UI state/action matrix. No Go API implementation starts until the P0 UI flow has a defined success path and negative path. If the official deadline is earlier than the current assumption, shrink to staging-only immediately. |
| Week 1 | Go API skeleton, React shell, identity, tenant-ready IDs, Firestore/Storage/Gemini spike, Project Registry, Repository/ProjectRepository schema, EnvironmentTopology and PromotionPolicy contracts, ProjectDocument registry and manifest validation. | Real Go service URL; create/list/detail/update/archive Project works; environment add/edit/retire semantics and required document states are represented by fixtures and API contracts; model and storage access are verified. |
| Week 2 | Project Workspace, Documents / AI Context view, Source Coverage preview, environment settings, architecture baseline, RAG retrieval with citations, deterministic cost calculator, decision record and role-specific report rendering. | One Project has a validated document baseline and a rebuildable Context Pack; one requirement produces a versioned Change Decision Brief and Agent Context Pack from the same DecisionRecord; missing data becomes 'NEEDS_INPUT', not invented content. |
| Week 3 | Agent Work Order, one developer-started Claude Code or Codex run, feature branch/PR, independent review evidence, Git/CI read-back, Cloud Build, Development/Testing evidence and Cloud Run staging. | A normal candidate reaches staging; out-of-scope paths, missing provenance, missing review or missing required evidence are blocked. |
| Week 4 | Staging verification, a 2–3 Change Release Bundle, isolated prod-demo promotion, release identity, config drift, source update invalidation, idempotency, negative tests and Release Receipt. | The same approved digest can be promoted only after every bundled Change passes its gate and the required human release gate; stale policy, target, digest or approval cannot release. |
| Week 5 | Baseline comparison, two holdout cases, second-person reproduction, 3–5 external role checks if available, cost/latency measurement, English submission material and video. | The second teammate can reproduce one positive and one negative case; the result is labelled with real evidence and limitations. Freeze feature work. |

## 4. P0 capability map

### P0-A — Project and source governance

- Project Registry, Workspace and Settings.
- Project lifecycle: 'DRAFT → ACTIVE → PAUSED → ARCHIVED'.
- Project update creates a new version and audit event.
- Archive stops new changes and promotion but preserves the decision ledger, evidence and receipts.
- Repository is a canonical source entity; ProjectRepository is a many-to-many association with role, path and confirmation state.
- Project Context Contract registry records the required document set, audience, authority, source version, effective time, access scope, index state and readiness reason; derived Context Pack always links back to its sources.
- Initial source onboarding uses a repository URL, handoffguard.yaml or a controlled fixture. It does not promise enterprise-wide GitHub/GitLab crawling.

### P0-B — Environment-aware change assurance

- Custom environment names and counts, standard types, order, target references, owners and evidence requirements.
- Versioned environment topology and promotion policy.
- Add, edit, reorder and retire without hard deletion.
- A change belongs to exactly one primary Project and references the relevant repository paths, services and environments.
- A Project setting change invalidates only the decisions and approvals that depend on the changed version; the invalidation is visible and auditable.

### P0-C — Evidence-aware RAG and decision support

- Versioned Markdown/TXT fixtures and at least one real embedding/retrieval path.
- Source citations, source hash, effective period, access scope and candidate/confirmed classification.
- Architecture options and cost drivers are candidates until a person accepts one.
- Cost arithmetic uses versioned price records and a deterministic calculator; unavailable inputs stay 'UNKNOWN'.
- A missing repository signal is reported as 'UNLINKED_REPOSITORY', 'UNKNOWN_DEPLOYMENT_SOURCE' or 'NEEDS_INPUT', never as a proven absence.

### P0-D — Agent handoff and release assurance

- An accepted DecisionRecord compiles into a versioned Agent Work Order.
- The work order includes repository, branch, allowed paths, forbidden actions, acceptance tests, required checks, target environment, expiry and source hash.
- One developer-started external coding-agent run is recorded; ContextRail does not host an arbitrary agent or store personal agent credentials.
- Git diff, PR, review, test, build, image digest, environment configuration and Cloud Run revision are cross-checked.
- A Release Bundle can group multiple Changes (feature, requirement change or bug fix) for one promotion; each Change keeps its own DecisionRecord and evidence, and the bundle cannot bypass a single Change gate.
- Human decisions remain separate: accept the requirement, accept the candidate, approve release.

## 5. Source discovery and coverage roadmap

The product can only discover private repositories inside an explicitly connected observation scope. A disconnected private GitLab instance is 'UNKNOWN', not an empty inventory.

### P0: evidence-bounded coverage check

P0 reconciles the sources that are already visible through fixtures, manifests, Cloud Build/Cloud Run evidence or a manually imported URL:

~~~text
Observed sources → canonical Repository → ProjectRepository mapping
      ↓                    ↓                    ↓
deployment/build       service/path          owner/role confirmation
~~~

The user can choose '建立新 Project', '加入既有 Project', '共用依賴', '指定 Monorepo path' or '標記為刻意未納管'.

### P1: one narrow provider adapter

After G4 passes and at least 40 person-hours of risk buffer remain, choose at most one:

- GitHub Organization read-only inventory;
- GitLab Group or self-hosted GitLab read-only inventory with configurable base URL;
- a fixed CI/collector adapter for a private network;
- a single Drive/Docs read-only source;
- a fixed remote-agent adapter, FDE Handoff Pack, ADC REST adapter or narrow Cloud Run revision rollback.

The choice must be based on the demo's largest evidence gap, not on the number of integrations we can list.

### P2: productisation

- Multiple provider connectors and scheduled reconciliation.
- Private-network/on-prem collector with enterprise identity and data residency controls.
- Fine-grained source ACL, retention/deletion workflows and organization-wide inventory.
- Multi-cloud, arbitrary Terraform parsing, autonomous FDE agent, arbitrary remote-agent orchestration and cross-resource rollback.

The P0 agent exit condition is one reproducible handoff chain with canonical IDs and evidence lineage—not a specific number of agents, an A2A endpoint or a chat transcript.

## 6. Milestones and release gates

| Gate | Meaning | If it fails |
| --- | --- | --- |
| G0 | Official deadline, capacity, accounts and source access are known. | Replan before implementation. |
| G1 | Go build, model, embedding, Firestore, Storage, identity and Cloud Run path work for real. | Fix the environment; do not claim cloud integration with mocks. |
| G2 | Project contract, environment policy, source citations, deterministic cost and DecisionRecord work. | Stop adding adapters; remove unsupported claims. |
| G3 | A real external coding-agent run produces a candidate and Git/CI evidence reaches staging. | Remove all stretch/P1 and describe the result as a narrower decision-to-deployment workflow. |
| G4 | Staging and isolated prod-demo promotion, drift checks, idempotency and receipt work. | Submit as staging-only; do not claim production readiness. |
| G5 | Reproduction, holdout, usability and baseline results are recorded. | Report limitations and keep market claims conditional. |

## 7. Definition of done for the prototype

The prototype is complete only when all of the following are observable:

1. A Project is created as a governance boundary and can reference multiple repositories without duplicating them.
2. An environment can be added, edited, reordered and retired; edits are versioned and dependent approvals become stale when required.
3. A private-source gap inside the connected observation scope is surfaced with evidence and a human resolution action.
4. A Project document baseline is validated, missing/stale/conflicting documents are visible, and a derived Context Pack can be rebuilt with source paths, versions, hashes and access scope.
5. One requirement produces a human-readable Change Decision Brief and an Agent Context Pack with the same 'decision_id', version, source hash and evidence references.
6. A real agent candidate is constrained by an immutable work order and checked against Git/CI evidence.
7. Staging and prod-demo use separate deployment evidence; same digest is necessary but not sufficient for promotion.
8. A Release Bundle can carry multiple Changes through one release attempt without losing per-Change scope, evidence or gate results.
9. A Release Receipt links requirement, decision, work order, candidate, review, commit, digest, target configuration, revision and human approval.
10. Negative cases block instead of silently passing: missing input, stale policy, unknown source, wrong path, missing review, wrong digest and target drift.
11. UI-01 through UI-19 in the [UI Flow & State Contract](docs/design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md) are executable or explicitly marked `NOT VERIFIED`; every mutation has a visible precondition, state transition, persistence result, actor, audit event and evidence reference, and the same state can be presented in `zh-TW` or `en` without translating data values.

Completion does not mean multi-tenant operations, production readiness, full private-network integration, autonomous FDE or a complete rollback system.

## 8. Decisions to confirm before implementation

These are the remaining decisions that can materially change the five-week plan:

- Official prototype/submission deadline and time zone.
- Actual weekly capacity for both people, not only the five-week calendar window.
- The first design-partner cohort: startup/SMB teams with the stated operating characteristics; enterprise validation is a later extension, not a P0 dependency.
- The demo source provider: GitHub, GitLab Cloud, self-hosted GitLab or a checked-in fixture.
- Whether the first demo uses one repo, a monorepo path or multiple repositories.
- The exact Cloud Run staging and prod-demo services and the identity allowed to deploy them.
- The first environment matrix and required promotion evidence; Development, Testing, Staging and Production are an example, not a forced global list.
- Which records must be retained after archive and the minimum access roles for the demo.
- Whether P1 should prioritize a source adapter, a fixed agent adapter, an FDE handoff pack, a Drive/Docs adapter, ADC or revision rollback.

Until these are confirmed, the safest implementation assumption is one synthetic Cloud Run application, one controlled repository source, four user-defined environments, staging as the real execution target and prod-demo as the isolated release target.

## 9. Related documents

- [Vision](vision.md)
- [Architecture](architecture.md)
- [Project Plan v4.2](docs/planning/CONTEXT_RAIL_PROJECT_PLAN_V4.2.zh-TW.md)
- [Critical Thinking Review](docs/reviews/CONTEXT_RAIL_CRITICAL_REVIEW.zh-TW.md)
- [UI Flow & State Contract](docs/design/CONTEXT_RAIL_UI_FLOW_STATE_CONTRACT.zh-TW.md)
- [Project Context Contract](docs/architecture/CONTEXT_RAIL_PROJECT_CONTEXT_CONTRACT.zh-TW.md)
- [Project Workspace mockup](output/ui/context-rail-project-workspace.html)
