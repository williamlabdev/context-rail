# ContextRail — submission deck outline (10 slides)

1. **Title** — ContextRail: environment-aware AI Change Assurance. One line: prove the change still matches the accepted intent before each governed environment. Team, Cloud Run URL, repo.
2. **The problem** — AI coding agents accelerate change; startup/SMB teams (5–30 engineers, Git + CI + AI assistant, ≥2 environments, no platform team) lose the answers to *what was accepted, did the agent stay in scope, is the deployed image the reviewed change*. Cost of a wrong promotion.
3. **The hero path** — the diagram from the README: contract → topology → decision pack → work order → candidate gate → release → receipt → prod-demo. Three human decisions, gates block but never approve.
4. **Contract before execution** — Project Context Contract, versioned topology (production read-only), document baseline with missing → readiness mapping, Context Pack that says PARTIAL instead of inventing. Screenshot: Documents / AI Context.
5. **Decision Pack** — NEEDS_INPUT with owners, Gemini as advisor (UNVERIFIED), one DecisionRecord → Brief + Agent Context Pack with the same lineage, hashed Work Order. Screenshot: dual view.
6. **Provenance over assertion** — candidate gate on observed evidence (GitHub read-back), allowed paths, independent review, single-operator waiver. Screenshot: BLOCKED with the offending path.
7. **Same digest is necessary, not sufficient** — release bundle gate, approval bound to the manifest hash, deployment verification (digest / target / config drift, smoke, idempotency), Release Receipt; prod-demo promotion with environment delta and chained receipts. Screenshot: receipt chain.
8. **Everything that moves goes STALE** — topology edit after approval, new candidate after receipt, document declared after decision; UI-14 / lineage drift screenshots.
9. **Built on Google Cloud** — Cloud Run (workspace + staging + prod-demo services), Cloud Build + Artifact Registry (digest-only promotion), Gemini advisor, `record-promotion.sh` reading live revisions with gcloud; Firestore as the next persistence step.
10. **Evidence and next steps** — every slice has DR-nnn + EB-nnn (tests, browser journeys, fixture-mutation hash, screenshots); 20/20 UI cases executable; boundaries (P0: no production execution, JSON state, actor recorded not authenticated); roadmap: Firestore, Cloud Run read-back inside the service, rollback stretch.

Appendix (if allowed): API surface table; evidence README of EB-009 (prod-demo) and EB-010 (Context Pack); Project Context Contract template.
