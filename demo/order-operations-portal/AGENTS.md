# Agent Instructions

## Allowed scope

For `REQ-001`, modify only the manual-review implementation, tests, UI and evidence files explicitly named by the work order. Keep the service synthetic and stateless.

## Required checks

Run `./scripts/validate-demo.sh`, `go test ./...` and `go build ./...`. Report the exact result and the commit/image identity available to you.

## Forbidden actions

- Do not add secrets or real customer data.
- Do not change `environments.md` to bypass the production block.
- Do not deploy to a production service, change IAM or alter payment/fulfillment behavior.
- Do not claim GitHub, CI or Cloud Run evidence that was not observed.
- Do not treat `docs/ai/context-pack.json` as a source of truth.

## Escalate

Stop and request human input if the request changes data retention, authorization, external integrations, environment policy or any path outside the approved work order.
