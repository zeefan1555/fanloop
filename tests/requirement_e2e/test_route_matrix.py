#!/usr/bin/env python3
import importlib.util
import pathlib
import tempfile
import unittest


SCRIPT = pathlib.Path(__file__).with_name("route_matrix.py")
SPEC = importlib.util.spec_from_file_location("route_matrix", SCRIPT)
route_matrix = importlib.util.module_from_spec(SPEC)
SPEC.loader.exec_module(route_matrix)


class RouteMatrixTest(unittest.TestCase):
    def test_discovers_every_status_route_alternative(self):
        current = {
            "context": {"step_id": "checks"},
            "available_routes": [
                {
                    "direction": "flow",
                    "when": {"any_of": [["unit_passed"], ["unit_failed"]]},
                    "route": {"next_step_id": "review"},
                },
                {
                    "direction": "loop",
                    "when": {"any_of": [["cr_failed"]]},
                    "route": {"back_step_id": "implement"},
                }
            ],
        }

        cases = route_matrix.route_cases(current)

        self.assertEqual(
            [(case["direction"], case["condition_ids"], case["target_step_id"]) for case in cases],
            [
                ("flow", ["unit_passed"], "review"),
                ("flow", ["unit_failed"], "review"),
                ("loop", ["cr_failed"], "implement"),
            ],
        )

    def test_builds_mock_values_for_every_public_output_type(self):
        specs = {
            "enum_value": {"type": "enum_value", "values": ["passed"]},
            "string": {"type": "string"},
            "boolean": {"type": "boolean"},
            "integer": {"type": "integer", "minimum": 2},
            "path": {"type": "path"},
            "url": {"type": "url"},
            "url_list": {"type": "url_list", "min_items": 2, "max_items": 3},
            "object": {"type": "object"},
        }
        with tempfile.TemporaryDirectory() as directory:
            root = pathlib.Path(directory)
            values = {
                output_type: route_matrix.mock_value(
                    "condition", spec, "step", "scenario", root
                )
                for output_type, spec in specs.items()
            }

            self.assertEqual(values["enum_value"], "passed")
            self.assertEqual(values["integer"], 2)
            self.assertEqual(len(values["url_list"]), 2)
            self.assertTrue((root / values["path"]).is_file())
            self.assertTrue(values["string"].startswith("e2e-agent-assertion:"))
            self.assertEqual(values["object"], {"e2e_mock": True})

    def test_selects_non_terminal_baseline_when_step_also_has_terminal_flows(self):
        flow_cases = [
            {
                "scenario_id": "flow-clarify-r01-a01",
                "expected_effect": "advanced",
                "target_step_id": "design",
            },
            {
                "scenario_id": "flow-clarify-r02-a01",
                "expected_effect": "completed",
                "target_step_id": "Done",
            },
        ]

        primary = route_matrix.select_baseline_flow_case(flow_cases)

        self.assertEqual(primary["scenario_id"], "flow-clarify-r01-a01")

    def test_rejects_multiple_non_terminal_baseline_targets(self):
        flow_cases = [
            {"expected_effect": "advanced", "target_step_id": "design"},
            {"expected_effect": "advanced", "target_step_id": "implement"},
        ]

        with self.assertRaisesRegex(AssertionError, "graph traversal"):
            route_matrix.select_baseline_flow_case(flow_cases)

    def test_printed_agent_request_matches_cli_arguments(self):
        case = {
            "scenario_id": "loop-checks-r01-a01",
            "direction": "loop",
            "source_step_id": "checks",
            "condition_ids": [
                "unit_tests_failed",
                "code_review_blocked",
                "code_review_document_published",
            ],
            "route": {"back_step_id": "implement"},
            "target_step_id": "implement",
        }
        results = [
            {
                "condition_id": "unit_tests_failed",
                "output": {"type": "enum_value", "value": "failed"},
            },
            {
                "condition_id": "code_review_blocked",
                "output": {"type": "enum_value", "value": "Block"},
            },
            {
                "condition_id": "code_review_document_published",
                "output": {
                    "type": "url",
                    "value": "https://bytedance.larkoffice.com/docx/Review",
                },
            },
        ]

        args, request = route_matrix.report_args(
            pathlib.Path("/tmp/e2e"), case, results
        )

        self.assertEqual(request["step_id"], "checks")
        self.assertEqual(request["condition_results"], results)
        self.assertEqual(request["route"], {"back_step_id": "implement"})
        self.assertIn("--back-step-id", args)
        self.assertEqual(args[args.index("--back-step-id") + 1], "implement")


if __name__ == "__main__":
    unittest.main()
