You are an independent technical reviewer. You did not write this change. You have read-only access to a clone of github.com/williamlabdev/order-operations-portal
at the candidate commit 8f39828a5412ec05c80064ba534cc3e878e307fd on branch agent/order-status-filter-2; the pull request is https://github.com/williamlabdev/order-operations-portal/pull/4.

ContextRail issued Work Order AWO-001 (hash sha256:9c42a69cdb15205d19208c19ef00d40e563a59f32a9babf85b9c6262377344fe):
Change: Filter the order list by status
Objective: Let operations staff list only the orders in one status, so the review queue (PENDING_REVIEW) can be read without scanning reviewed orders.
Acceptance criteria:
- GET /api/orders?status=<status> returns only the orders whose status equals <status>
- GET /api/orders without a status parameter returns every order, as before
- a status value that is not PENDING_REVIEW, APPROVED or REJECTED is answered with 400 and a JSON error body
- main_test.go covers the three cases and go test ./... passes
Allowed paths: main.go, main_test.go
Forbidden: deploy:production, modify:iam, read:deployment-secrets, merge-or-approve-release, change-scope-without-new-decision-version, authentication, production deployment, changing the order review endpoint, adding dependencies

The pull request diff against main:
```diff
diff --git a/main.go b/main.go
index ea6ef57..6ae3e7d 100644
--- a/main.go
+++ b/main.go
@@ -200,11 +200,39 @@ func listenAddr(port string) string {
 	return ":8080"
 }
 
+// orderStatuses are the only values an order's status can take: the initial
+// PENDING_REVIEW and the two review decisions.
+var orderStatuses = []string{"PENDING_REVIEW", "APPROVED", "REJECTED"}
+
+// validOrderStatus reports whether raw is one of orderStatuses, compared
+// exactly (no trimming or case folding) so the filter matches the stored value.
+func validOrderStatus(raw string) bool {
+	for _, status := range orderStatuses {
+		if raw == status {
+			return true
+		}
+	}
+	return false
+}
+
+// listOrders returns every order, or only the orders in one status when the
+// `status` query parameter is present (AWO-001). A status parameter that is
+// not one of orderStatuses is a 400 with a JSON error; no parameter at all
+// keeps the previous behaviour of listing everything.
 func (s *server) listOrders(w http.ResponseWriter, r *http.Request) {
+	query := r.URL.Query()
+	filter := query.Get("status")
+	if query.Has("status") && !validOrderStatus(filter) {
+		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "status must be " + strings.Join(orderStatuses, ", ")})
+		return
+	}
 	s.mu.RLock()
 	defer s.mu.RUnlock()
 	result := make([]orderView, 0, len(s.orders))
 	for _, item := range s.orders {
+		if filter != "" && item.Status != filter {
+			continue
+		}
 		result = append(result, orderView{order: *item, Evidence: s.evidenceOf(item.ID)})
 	}
 	writeJSON(w, http.StatusOK, result)
diff --git a/main_test.go b/main_test.go
index 593c6be..a7b93fb 100644
--- a/main_test.go
+++ b/main_test.go
@@ -273,6 +273,129 @@ func TestEvidenceIsListedWithTheOrder(t *testing.T) {
 	}
 }
 
+// --- AWO-001: filter the order list by status ----------------------------------
+
+// listOrders performs GET /api/orders with the given raw query (may be empty)
+// and returns the response plus the decoded order list when it is a 200.
+func listOrders(t *testing.T, s *server, rawQuery string) (*httptest.ResponseRecorder, []orderView) {
+	t.Helper()
+	target := "/api/orders"
+	if rawQuery != "" {
+		target += "?" + rawQuery
+	}
+	req := httptest.NewRequest(http.MethodGet, target, nil)
+	res := httptest.NewRecorder()
+	s.routes().ServeHTTP(res, req)
+	if res.Code != http.StatusOK {
+		return res, nil
+	}
+	var orders []orderView
+	if err := json.NewDecoder(res.Body).Decode(&orders); err != nil {
+		t.Fatalf("GET %s: decode orders: %v", target, err)
+	}
+	return res, orders
+}
+
+// reviewOrder approves or rejects one order so the fixture has mixed statuses.
+func reviewOrder(t *testing.T, s *server, orderID, decision string) {
+	t.Helper()
+	body, _ := json.Marshal(reviewRequest{Decision: decision, Note: "reviewed for the status filter test"})
+	req := httptest.NewRequest(http.MethodPost, "/api/orders/"+orderID+"/review", bytes.NewReader(body))
+	res := httptest.NewRecorder()
+	s.routes().ServeHTTP(res, req)
+	if res.Code != http.StatusOK {
+		t.Fatalf("review %s as %s: got %d, want 200: %s", orderID, decision, res.Code, res.Body.String())
+	}
+}
+
+// GET /api/orders?status=<status> returns only the orders in that status, for
+// each of the three statuses, and a status nobody is in yields an empty array.
+func TestListOrdersFiltersByStatus(t *testing.T) {
+	s := newServer()
+	reviewOrder(t, s, "ORD-1002", "APPROVED")
+	want := map[string][]string{
+		"PENDING_REVIEW": {"ORD-1001"},
+		"APPROVED":       {"ORD-1002"},
+		"REJECTED":       {},
+	}
+	for status, ids := range want {
+		res, orders := listOrders(t, s, "status="+status)
+		if res.Code != http.StatusOK {
+			t.Fatalf("status=%s: got %d, want 200: %s", status, res.Code, res.Body.String())
+		}
+		if orders == nil {
+			t.Fatalf("status=%s: response must be a JSON array even when empty", status)
+		}
+		if len(orders) != len(ids) {
+			t.Fatalf("status=%s: got %d orders, want %d: %+v", status, len(orders), len(ids), orders)
+		}
+		for i, item := range orders {
+			if item.Status != status {
+				t.Fatalf("status=%s: order %s has status %s", status, item.ID, item.Status)
+			}
+			if item.ID != ids[i] {
+				t.Fatalf("status=%s: got order %s, want %s", status, item.ID, ids[i])
+			}
+			if item.Evidence == nil {
+				t.Fatalf("status=%s: filtered order %s must still carry its evidence array", status, item.ID)
+			}
+		}
+	}
+	// Reviewing again moves the order between filters.
+	reviewOrder(t, s, "ORD-1002", "REJECTED")
+	if _, orders := listOrders(t, s, "status=REJECTED"); len(orders) != 1 || orders[0].ID != "ORD-1002" {
+		t.Fatalf("status=REJECTED after re-review: %+v", orders)
+	}
+	if _, orders := listOrders(t, s, "status=APPROVED"); len(orders) != 0 {
+		t.Fatalf("status=APPROVED after re-review must be empty: %+v", orders)
+	}
+}
+
+// GET /api/orders without a status parameter still returns every order.
+func TestListOrdersWithoutStatusReturnsAll(t *testing.T) {
+	s := newServer()
+	reviewOrder(t, s, "ORD-1001", "REJECTED")
+	res, orders := listOrders(t, s, "")
+	if res.Code != http.StatusOK {
+		t.Fatalf("got %d, want 200", res.Code)
+	}
+	if len(orders) != len(s.orders) {
+		t.Fatalf("got %d orders, want all %d: %+v", len(orders), len(s.orders), orders)
+	}
+	seen := map[string]string{}
+	for _, item := range orders {
+		seen[item.ID] = item.Status
+	}
+	if seen["ORD-1001"] != "REJECTED" || seen["ORD-1002"] != "PENDING_REVIEW" {
+		t.Fatalf("unfiltered list must include orders of every status: %v", seen)
+	}
+	// Unrelated query parameters do not turn the filter on.
+	if _, orders := listOrders(t, s, "page=1"); len(orders) != len(s.orders) {
+		t.Fatalf("unrelated query parameter must not filter: %+v", orders)
+	}
+}
+
+// A status value outside PENDING_REVIEW, APPROVED and REJECTED is a 400 with a
+// JSON error body; the comparison is exact, so case and whitespace variants
+// are rejected rather than silently matching nothing.
+func TestListOrdersRejectsUnknownStatus(t *testing.T) {
+	for _, rawQuery := range []string{
+		"status=SHIPPED",
+		"status=pending_review",
+		"status=%20APPROVED",
+		"status=",
+	} {
+		s := newServer()
+		res, _ := listOrders(t, s, rawQuery)
+		if res.Code != http.StatusBadRequest {
+			t.Fatalf("%s: got %d, want 400: %s", rawQuery, res.Code, res.Body.String())
+		}
+		if msg := decodeError(t, res); !strings.Contains(msg, "status") {
+			t.Fatalf("%s: error must name the status parameter: %q", rawQuery, msg)
+		}
+	}
+}
+
 // The unknown action under /api/orders/{id}/ stays a 404 and the review
 // endpoint is unchanged by the new dispatcher (AC-005).
 func TestUnknownOrderActionIsNotFound(t *testing.T) {
```

Review the change against the Work Order: does it meet every acceptance criterion, stay inside the allowed paths, keep the existing
behaviour and tests intact, and test what it adds? Read the files you need. Do not suggest style changes that do not affect correctness.

End your reply with exactly one line of JSON and nothing after it:
{"verdict": "APPROVED" or "CHANGES_REQUESTED", "reasons": ["..."]}
