You are working in a clone of github.com/williamlabdev/order-operations-portal, on branch agent/order-status-filter (based on main).
ContextRail issued Work Order AWO-001 (hash sha256:8a4a7dfcad67e12ccd8d985892e33556a124d8148b33987a8176645c81f53011) for this change. Do exactly this change and nothing else.

Change: Filter the order list by status
Objective: Let operations staff list only the orders in one status, so the review queue (PENDING_REVIEW) can be read without scanning reviewed orders.

Acceptance criteria:
- GET /api/orders?status=<status> returns only the orders whose status equals <status>
- GET /api/orders without a status parameter returns every order, as before
- a status value that is not PENDING_REVIEW, APPROVED or REJECTED is answered with 400 and a JSON error body
- main_test.go covers the three cases and go test ./... passes

In scope: status query parameter on GET /api/orders, validation of the status value, unit tests
Out of scope: authentication, order review behaviour, web UI, new dependencies, production environment

Rules of this Work Order:
- You may edit only these paths: main.go, main_test.go. Editing any other file fails the run.
- Forbidden: deploy:production, modify:iam, read:deployment-secrets, merge-or-approve-release, change-scope-without-new-decision-version, authentication, production deployment, changing the order review endpoint, adding dependencies.
- Do not run git and do not try to push, commit or open a pull request; the harness does that after you stop.
- Read AGENTS.md and docs/engineering/development.md first and follow them.
- The required checks on the pull request include the CI jobs "test" (go test ./...) and "build" (go build and scripts/validate-demo.sh).
  Run go vet ./... and go test ./... yourself before you stop, and gofmt the files you changed.

When you are done, reply with a short summary: which files you changed, what you added, and the exact test commands you ran with their result.
