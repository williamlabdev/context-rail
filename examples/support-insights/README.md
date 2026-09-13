# Support Insights

Support Insights is the second template consumer fixture. It models a private GitHub Python analytics Project with a read-only Cloud Run staging target. It contains no real support tickets or customer data.

The purpose of this fixture is to verify that the Project Context Contract is not coupled to the Go Order Operations Portal runtime.

## Validation

From the ContextRail workspace:

```sh
template/.venv/bin/python template/scripts/validate_project_context.py --root examples/support-insights --strict
template/.venv/bin/python template/scripts/rebuild_context_pack.py --root examples/support-insights
```

Production is represented as a human-gated protected target and is not deployed.

The synthetic report itself can be verified with `python -m unittest discover -s tests`, then viewed with `python -m src.report`.
