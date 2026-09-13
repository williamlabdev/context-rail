from __future__ import annotations

import json
from collections import Counter
from pathlib import Path


FIXTURE = Path(__file__).resolve().parents[1] / "fixtures" / "tickets.json"


def build_report(path: Path = FIXTURE) -> dict[str, object]:
    tickets = json.loads(path.read_text(encoding="utf-8"))
    if not isinstance(tickets, list) or not tickets:
        raise ValueError("ticket fixture must be a non-empty list")
    themes = Counter(ticket["theme"] for ticket in tickets)
    average_hours = sum(ticket["response_hours"] for ticket in tickets) / len(tickets)
    return {
        "period": "2026-W37",
        "ticket_count": len(tickets),
        "themes": dict(sorted(themes.items())),
        "average_response_hours": round(average_hours, 2),
        "source": "fixtures/tickets.json",
    }


if __name__ == "__main__":
    print(json.dumps(build_report(), indent=2))
