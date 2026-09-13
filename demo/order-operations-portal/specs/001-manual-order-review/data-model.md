# Data Model: Manual Order Review

## Order

Represents one synthetic order exception shown to operations staff.

| Field | Type | Required | Constraint |
| --- | --- | --- | --- |
| `id` | string | yes | unique within the process; example `ORD-1001` |
| `customer` | string | yes | synthetic display label only |
| `amount` | integer | yes | synthetic whole-number amount for display |
| `reason` | string | yes | explains why review is needed |
| `status` | enum | yes | `PENDING_REVIEW`, `APPROVED` or `REJECTED` |
| `note` | string | no | trimmed review reason after a decision |

## Review Request

Represents the operator's explicit decision for one order.

| Field | Type | Required | Constraint |
| --- | --- | --- | --- |
| `decision` | enum | yes | exactly `APPROVED` or `REJECTED` |
| `note` | string | yes | must contain a non-whitespace character |

## State transition

```text
PENDING_REVIEW --(APPROVED + note)--> APPROVED
PENDING_REVIEW --(REJECTED + note)--> REJECTED
APPROVED --(valid review + note)--> APPROVED or REJECTED
REJECTED --(valid review + note)--> APPROVED or REJECTED
```

The latest valid review replaces the previous status and note. The prototype keeps state in process memory. Restarting the service resets the seeded orders; persistence is outside this Change Slice.
