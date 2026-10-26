# EB-019 AI review, run 2

- reviewer: separate headless `claude -p --bare` run, model `claude-opus-5-5`, `--tools Read` only, permission mode dontAsk
- reviewed: https://github.com/williamlabdev/order-operations-portal/pull/4 at 8f39828a5412ec05c80064ba534cc3e878e307fd
- at: 2026-10-26T15:54:37Z
- cost: US$0.1487884
- transcript: `review-transcript-2.jsonl`

## Verdict: APPROVED

- GET /api/orders?status=<status> returns only orders whose status exactly equals the parameter, each still with its evidence array
- With no status parameter every order is returned as before; parameters other than status do not filter
- A status that is present but not exactly PENDING_REVIEW, APPROVED or REJECTED (including empty, lowercase or with a leading space) gets a 400 with a JSON error body
- main_test.go adds three behavioural tests for the three cases; read-through shows they compile and are not flaky (each filtered result has at most one order), though go test was not run
- Only main.go and main_test.go changed; the review endpoint, routing, authentication, dependencies and existing tests are untouched
- Non-blocking: query pairs Go cannot parse (e.g. status=%ZZ) are silently dropped by URL.Query(), so they list every order instead of returning 400

## Full reply

I recommend approving this pull request. All four acceptance criteria are met, the diff only touches `main.go` and `main_test.go`, and existing behaviour is unchanged. I couldn't run `go test ./...` with read-only access, so I checked by reading the code that it compiles and passes.

**Acceptance criteria**
1. **Filter works.** `GET /api/orders?status=X` only appends orders whose `Status == X`. Each order still carries its evidence array, so filtered results look the same as before.
2. **No parameter means everything.** The filter only runs when a non-empty status has passed validation. With no `status` parameter, nothing is filtered and the old loop runs as before. Other parameters like `page=1` don't trigger the filter.
3. **Bad status gets a 400 with a JSON error.** If `status` is present but isn't exactly one of `PENDING_REVIEW`, `APPROVED` or `REJECTED`, `writeJSON` returns a 400 with `{"error": "status must be …"}`. That includes an empty value, lowercase, or a leading space.
4. **Tests cover the three cases.** Each test checks real behaviour:
   - `TestListOrdersFiltersByStatus` covers all three statuses, the empty-array case and an order moving between filters after a second review.
   - `TestListOrdersWithoutStatusReturnsAll` checks that orders of mixed statuses all come back, and that `page=1` doesn't filter.
   - `TestListOrdersRejectsUnknownStatus` checks four bad values. It uses the existing `decodeError`, which also checks that the response is JSON.

**Why I expect it to compile and pass**
- The test helper `listOrders` is a plain function and the handler is a method on `server`, so the shared name is legal Go. No other function in the package uses the names `listOrders` or `reviewOrder`.
- `url.Values.Has` needs Go 1.17 and the module declares 1.22. `strings` is already imported in both files.
- The orders are stored in a map, so their order is random. The test compares by position (`ids[i]`), but every filtered result has at most one order, so the random order can't make it flaky.
- A response of `[]` decodes into a non-nil empty slice, so the "must be a JSON array" check holds for `REJECTED`.

**Scope**
- Nothing in the review endpoint, routing, `orderAction` or authentication changed. No dependencies were added and no existing tests were changed.

**Non-blocking note**
- Go's `r.URL.Query()` silently drops query pairs it can't parse, such as `?status=%ZZ` or a pair containing `;`. In that case the server sees no status parameter and returns every order with a 200, instead of a 400. This is an edge case outside the acceptance criteria as written, but worth a follow-up so a mistyped filter isn't mistaken for a filtered list.

{"verdict": "APPROVED", "reasons": ["GET /api/orders?status=<status> returns only orders whose status exactly equals the parameter, each still with its evidence array", "With no status parameter every order is returned as before; parameters other than status do not filter", "A status that is present but not exactly PENDING_REVIEW, APPROVED or REJECTED (including empty, lowercase or with a leading space) gets a 400 with a JSON error body", "main_test.go adds three behavioural tests for the three cases; read-through shows they compile and are not flaky (each filtered result has at most one order), though go test was not run", "Only main.go and main_test.go changed; the review endpoint, routing, authentication, dependencies and existing tests are untouched", "Non-blocking: query pairs Go cannot parse (e.g. status=%ZZ) are silently dropped by URL.Query(), so they list every order instead of returning 400"]}
