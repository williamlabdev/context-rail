# ContextRail — demo video script (target 2:45, hard limit 3:00)

Format: screen recording of the deployed workspace (Cloud Run URL) with voice-over. English narration; the workspace can be shown in zh-TW for one beat to prove UI-16. Every beat below is something the browser journeys already verify, so the recording follows `tests/browser/*.spec.ts` step by step.

| # | Time | Screen | Voice-over (draft) |
| --- | --- | --- | --- |
| 1 | 0:00–0:15 | Title card → workspace header on the Cloud Run URL | "AI coding agents ship changes fast. ContextRail is the control plane that proves, before each governed environment, that the change still matches what was accepted — and blocks it when it doesn't." |
| 2 | 0:15–0:35 | Project Registry → Order Operations Portal; Environment Topology with development / testing / staging / prod-demo / production (production read-only) | "A Project declares its context contract and its promotion path. Every topology change is an immutable version; production stays read-only in this prototype." |
| 3 | 0:35–1:00 | Changes → New Change with missing inputs → NEEDS_INPUT table naming owners → supply inputs → DECISION_READY → Accept decision (Gemini advisor options visible) | "A change request is evaluated deterministically. Missing inputs name the person who must supply them — the system never invents them. Gemini proposes options; a human decides." |
| 4 | 1:00–1:15 | Brief and Agent Context Pack side by side with the same lineage strip; Compile Work Order (hash) | "One DecisionRecord produces two views: a Brief for people and a Context Pack for the agent — same decision id, same source snapshot hash. The Work Order is hashed." |
| 5 | 1:15–1:40 | Candidate submitted with an out-of-scope path → BLOCKED with the path; compliant candidate → CANDIDATE_ACCEPTABLE → Accept for promotion | "The coding agent's result is checked against the Work Order: allowed paths, required checks, independent review. The gate blocks; it never approves. The second human decision accepts the candidate." |
| 6 | 1:40–2:10 | Release: bundle gate (one blocked change blocks the bundle) → build digest → approval bound to manifest hash → deployment record → Release Receipt | "A release bundles accepted candidates. The approval is bound to the manifest hash. The deployment record is verified: digest, target, configuration, smoke — and the Release Receipt links everything back to the decision." |
| 7 | 2:10–2:35 | Promote to prod-demo: environment delta table → approval → staging revision refused (`new_revision`) → new revision → chained receipt (`previous_receipt_id`) | "The same digest goes to prod-demo. The environment differences are listed and bound by a new approval; re-using the staging revision is refused; the second receipt chains to the first." |
| 8 | 2:35–2:50 | Edit staging target after approval → release STALE; Documents: declare a missing runbook → decision NEEDS_INPUT → Context Pack PARTIAL | "When anything moves — the environment, the sources, a document — dependent decisions go stale instead of being reused silently." |
| 9 | 2:50–3:00 | Close: architecture strip (Cloud Run · Cloud Build · Artifact Registry · Gemini), repo + evidence folder | "Go on Cloud Run, built by Cloud Build, promoted by digest, advised by Gemini. Every slice of ContextRail itself ships with a DecisionRecord and an Evidence Bundle." |

## Recording checklist

- Fresh `CONTEXT_RAIL_STATE_DIR` on the deployed service (or a scripted seed via the API) so ids read REL-001 / RR-001 in order.
- Seed the change up to the accepted candidate through the API (`tests/browser/prod-demo-promotion.spec.ts` shows the calls) to keep beats 3–5 under 65 seconds; type only what the viewer must see.
- Show the fixture-mutation check once (evidence README) if time allows; otherwise mention it in the closing line.
- Locale: switch to 繁中 for beat 2 or 8 for one sentence, then back to EN.
- Keep the browser at 1440×900; hide bookmarks bar.
