# HTTP Contract: Manual Order Review

Base URL: `http://localhost:8080`

## Health

`GET /healthz`

- Success: `200` with a JSON object whose `status` is `ok`.
- Purpose: local and staging smoke check only.

## List orders

`GET /api/orders`

- Success: `200` with a JSON array containing the seeded orders.
- Each order includes its ID, display information, reason and current status.

## Review an order

`POST /api/orders/{order_id}/review`

Request body:

```json
{
  "decision": "APPROVED",
  "note": "Address confirmed"
}
```

Success:

- `200` with the updated order and a UTC `reviewedAt` timestamp.

Validation and lookup failures:

- `400` for invalid JSON, an unsupported decision or an empty/whitespace note.
- `404` when the order ID does not exist.

The contract has no authentication, persistence, payment, fulfillment or production side effect. Those capabilities require a new Request and DecisionRecord.
