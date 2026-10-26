You are an independent technical reviewer. You did not write this change. You have read-only access to a clone of github.com/williamlabdev/order-operations-portal
at the candidate commit b646caf96b287eaa8593856c34f28208a8d66906 on branch agent/order-status-filter; the pull request is https://github.com/williamlabdev/order-operations-portal/pull/3.

ContextRail issued Work Order AWO-001 (hash sha256:8a4a7dfcad67e12ccd8d985892e33556a124d8148b33987a8176645c81f53011):
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
index ea6ef57..830169b 100644
--- a/main.go
+++ b/main.go
@@ -200,16 +200,50 @@ func listenAddr(port string) string {
 	return ":8080"
 }
 
+// orderStatuses are the only values an order can carry, so they are the only
+// values the status filter on GET /api/orders accepts.
+var orderStatuses = map[string]bool{"PENDING_REVIEW": true, "APPROVED": true, "REJECTED": true}
+
+// listOrders returns every order, or only the orders in one status when the
+// request carries ?status=<status> (AWO-001). A status that is not one of
+// orderStatuses is a 400 with a JSON error rather than an empty list, so a
+// typo in the queue filter cannot look like an empty queue.
 func (s *server) listOrders(w http.ResponseWriter, r *http.Request) {
+	status, filtered, err := statusFilter(r.URL.Query())
+	if err != nil {
+		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
+		return
+	}
 	s.mu.RLock()
 	defer s.mu.RUnlock()
 	result := make([]orderView, 0, len(s.orders))
 	for _, item := range s.orders {
+		if filtered && item.Status != status {
+			continue
+		}
 		result = append(result, orderView{order: *item, Evidence: s.evidenceOf(item.ID)})
 	}
 	writeJSON(w, http.StatusOK, result)
 }
 
+// statusFilter reads the optional status query parameter. It reports whether
+// a filter was requested and, if so, which status; an absent parameter means
+// no filter, and a present parameter must be given once and be exactly one
+// of orderStatuses (no trimming or case folding, matching the stored value).
+func statusFilter(query url.Values) (status string, filtered bool, err error) {
+	values, present := query["status"]
+	if !present {
+		return "", false, nil
+	}
+	if len(values) != 1 {
+		return "", false, fmt.Errorf("status must be given once")
+	}
+	if !orderStatuses[values[0]] {
+		return "", false, fmt.Errorf("status must be PENDING_REVIEW, APPROVED or REJECTED")
+	}
+	return values[0], true, nil
+}
+
 // evidenceOf returns a copy of the order's references, never nil, so the JSON
 // always carries an `evidence` array. Callers hold s.mu.
 func (s *server) evidenceOf(orderID string) []evidenceReference {
diff --git a/main_test.go b/main_test.go
index 593c6be..034173f 100644
--- a/main_test.go
+++ b/main_test.go
@@ -27,6 +27,125 @@ func TestHealthAndOrderList(t *testing.T) {
 	}
 }
 
+// --- AWO-001: filter the order list by status --------------------------------
+
+// listOrders issues GET /api/orders with the given raw query (may be empty)
+// and returns the response.
+func listOrders(t *testing.T, s *server, rawQuery string) *httptest.ResponseRecorder {
+	t.Helper()
+	target := "/api/orders"
+	if rawQuery != "" {
+		target += "?" + rawQuery
+	}
+	req := httptest.NewRequest(http.MethodGet, target, nil)
+	res := httptest.NewRecorder()
+	s.routes().ServeHTTP(res, req)
+	return res
+}
+
+// decodeOrders decodes a 200 order list into id → status.
+func decodeOrders(t *testing.T, res *httptest.ResponseRecorder) map[string]string {
+	t.Helper()
+	if res.Code != http.StatusOK {
+		t.Fatalf("got %d, want 200: %s", res.Code, res.Body.String())
+	}
+	var orders []struct {
+		ID     string `json:"id"`
+		Status string `json:"status"`
+	}
+	if err := json.NewDecoder(res.Body).Decode(&orders); err != nil {
+		t.Fatalf("decode orders: %v", err)
+	}
+	statuses := make(map[string]string, len(orders))
+	for _, item := range orders {
+		statuses[item.ID] = item.Status
+	}
+	return statuses
+}
+
+// reviewOrder moves an order out of PENDING_REVIEW so the list has mixed
+// statuses to filter on.
+func reviewOrder(t *testing.T, s *server, orderID, decision string) {
+	t.Helper()
+	body, _ := json.Marshal(reviewRequest{Decision: decision, Note: "reviewed for the filter test"})
+	req := httptest.NewRequest(http.MethodPost, "/api/orders/"+orderID+"/review", bytes.NewReader(body))
+	res := httptest.NewRecorder()
+	s.routes().ServeHTTP(res, req)
+	if res.Code != http.StatusOK {
+		t.Fatalf("review %s as %s: got %d, want 200", orderID, decision, res.Code)
+	}
+}
+
+// GET /api/orders?status=<status> returns only the orders in that status,
+// for each of the three statuses.
+func TestListOrdersFiltersByStatus(t *testing.T) {
+	s := newServer()
+	reviewOrder(t, s, "ORD-1001", "APPROVED")
+	// ORD-1002 stays PENDING_REVIEW.
+
+	cases := map[string]map[string]string{
+		"PENDING_REVIEW": {"ORD-1002": "PENDING_REVIEW"},
+		"APPROVED":       {"ORD-1001": "APPROVED"},
+		"REJECTED":       {},
+	}
+	for status, want := range cases {
+		got := decodeOrders(t, listOrders(t, s, "status="+status))
+		if len(got) != len(want) {
+			t.Fatalf("status=%s: got %v, want %v", status, got, want)
+		}
+		for id, wantStatus := range want {
+			if got[id] != wantStatus {
+				t.Fatalf("status=%s: got %v, want %v", status, got, want)
+			}
+		}
+	}
+
+	// A review that changes the status moves the order between filters.
+	reviewOrder(t, s, "ORD-1002", "REJECTED")
+	if got := decodeOrders(t, listOrders(t, s, "status=PENDING_REVIEW")); len(got) != 0 {
+		t.Fatalf("queue must be empty after every order is reviewed: %v", got)
+	}
+	if got := decodeOrders(t, listOrders(t, s, "status=REJECTED")); len(got) != 1 || got["ORD-1002"] != "REJECTED" {
+		t.Fatalf("rejected filter must list the rejected order: %v", got)
+	}
+}
+
+// GET /api/orders without a status parameter still returns every order,
+// whatever their statuses; a different, unrelated query parameter does not
+// filter either.
+func TestListOrdersWithoutStatusReturnsAll(t *testing.T) {
+	s := newServer()
+	reviewOrder(t, s, "ORD-1001", "REJECTED")
+	for _, rawQuery := range []string{"", "customer=Acme"} {
+		got := decodeOrders(t, listOrders(t, s, rawQuery))
+		if len(got) != 2 || got["ORD-1001"] != "REJECTED" || got["ORD-1002"] != "PENDING_REVIEW" {
+			t.Fatalf("query %q: got %v, want both orders", rawQuery, got)
+		}
+	}
+}
+
+// A status that is not PENDING_REVIEW, APPROVED or REJECTED is a 400 with a
+// JSON error body, never an empty list.
+func TestListOrdersRejectsUnknownStatus(t *testing.T) {
+	s := newServer()
+	for _, rawQuery := range []string{
+		"status=SHIPPED",
+		"status=approved",
+		"status=PENDING_REVIEW%20",
+		"status=",
+		"status",
+		"status=PENDING_REVIEW&status=APPROVED",
+	} {
+		res := listOrders(t, s, rawQuery)
+		if res.Code != http.StatusBadRequest {
+			t.Fatalf("query %q: got %d, want 400: %s", rawQuery, res.Code, res.Body.String())
+		}
+		if msg := decodeError(t, res); !strings.Contains(msg, "status") {
+			t.Fatalf("query %q: error must name the status parameter: %q", rawQuery, msg)
+		}
+	}
+}
+
 func TestReviewRequiresNote(t *testing.T) {
 	for _, note := range []string{"", " \t\n ", "　"} {
 		s := newServer()
```

Review the change against the Work Order: does it meet every acceptance criterion, stay inside the allowed paths, keep the existing
behaviour and tests intact, and test what it adds? Read the files you need. Do not suggest style changes that do not affect correctness.

End your reply with exactly one line of JSON and nothing after it:
{"verdict": "APPROVED" or "CHANGES_REQUESTED", "reasons": ["..."]}
