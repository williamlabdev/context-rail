import unittest

from src.report import build_report


class ReportTests(unittest.TestCase):
    def test_report_is_reproducible_from_synthetic_fixture(self) -> None:
        report = build_report()
        self.assertEqual(report["ticket_count"], 3)
        self.assertEqual(report["themes"], {"access": 2, "billing": 1})
        self.assertEqual(report["average_response_hours"], 5.33)


if __name__ == "__main__":
    unittest.main()
