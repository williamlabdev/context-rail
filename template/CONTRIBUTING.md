# Contributing

Every change starts as a Request. Before implementation, record its scope, acceptance criteria, affected architecture and human decision. Keep generated Context Pack content separate from source documents.

Pull requests must include reproducible test/build evidence and identify the reviewer. Changes to environment policy, protected paths, data classification or approval roles need explicit human review.

Run the repository's validation from the declared Python virtual environment before opening a pull request:

```sh
python3 -m venv .venv
.venv/bin/python -m pip install -r requirements.txt
.venv/bin/python scripts/validate_project_context.py
```

Do not add real credentials, private customer data or unverified deployment claims. Use `SECURITY.md` for sensitive vulnerability reports.
