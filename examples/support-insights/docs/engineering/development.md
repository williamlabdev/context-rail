# Development Contract

## Prerequisites

- Python 3.11 or newer.
- A project virtual environment for any future dependencies.
- Synthetic fixtures only.

## Commands

```sh
python -m unittest discover -s tests
python -m compileall src
python -m src.report
```

The report must be reproducible from the checked-in fixture and must fail clearly when the fixture is incomplete.
