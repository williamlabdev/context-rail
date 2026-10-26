You are working in a clone of {repository}, on branch {target_branch} (based on {base_branch}).
ContextRail issued Work Order {work_order_id} (hash {work_order_hash}) for this change. Do exactly this change and nothing else.

Change: {title}
Objective: {objective}

Acceptance criteria:
{acceptance}

In scope: {scope_included}
Out of scope: {scope_excluded}

Rules of this Work Order:
- You may edit only these paths: {allowed_paths}. Editing any other file fails the run.
- Forbidden: {forbidden_actions}.
- Do not run git and do not try to push, commit or open a pull request; the harness does that after you stop.
- Read AGENTS.md and docs/engineering/development.md first and follow them.
- The required checks on the pull request include the CI jobs "test" (go test ./...) and "build" (go build and scripts/validate-demo.sh).
  Run go vet ./... and go test ./... yourself before you stop, and gofmt the files you changed.

When you are done, reply with a short summary: which files you changed, what you added, and the exact test commands you ran with their result.
