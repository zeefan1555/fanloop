#!/usr/bin/env python3
"""Execute every Flow/Loop Route alternative exposed by Commonloop flow status."""

from __future__ import annotations

import argparse
import datetime as dt
import hashlib
import json
import os
import pathlib
import shlex
import shutil
import subprocess
import sys
import time
import traceback
from typing import Any


TRACE_DOCUMENT_URL = "https://bytedance.larkoffice.com/docx/CommonloopRouteMatrixE2E"
CLI_LOG_DOCUMENT_URL = "https://bytedance.larkoffice.com/docx/CommonloopRouteMatrixCLILogE2E"


def now() -> str:
    return dt.datetime.now(dt.timezone.utc).isoformat()


def write_json(path: pathlib.Path, value: Any) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(json.dumps(value, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")


def read_json(path: pathlib.Path) -> dict[str, Any]:
    return json.loads(path.read_text(encoding="utf-8"))


def current(status: dict[str, Any]) -> dict[str, Any] | None:
    state = status["data"]["state"]
    return None if state["status"] == "completed" else state["current"]


def route_cases(current_state: dict[str, Any]) -> list[dict[str, Any]]:
    step_id = current_state["context"]["step_id"]
    cases: list[dict[str, Any]] = []
    route_indexes = {"flow": 0, "loop": 0}
    for route in current_state["available_routes"]:
        direction = route["direction"]
        route_indexes[direction] += 1
        route_index = route_indexes[direction]
        selection = route["route"]
        target = selection.get("next_step_id") or selection.get("back_step_id") or "Done"
        effect = "completed" if selection.get("terminal") else (
            "advanced" if direction == "flow" else "looped"
        )
        for alternative_index, condition_ids in enumerate(route["when"]["any_of"], start=1):
            cases.append(
                {
                    "scenario_id": f"{direction}-{step_id}-r{route_index:02d}-a{alternative_index:02d}",
                    "direction": direction,
                    "source_step_id": step_id,
                    "route_index": route_index,
                    "alternative_index": alternative_index,
                    "condition_ids": condition_ids,
                    "route": selection,
                    "target_step_id": target,
                    "expected_effect": effect,
                }
            )
    return cases


def select_baseline_flow_case(flow_cases: list[dict[str, Any]]) -> dict[str, Any]:
    non_terminal = [case for case in flow_cases if case["expected_effect"] != "completed"]
    if non_terminal:
        if len({case["target_step_id"] for case in non_terminal}) != 1:
            raise AssertionError("Step has divergent non-terminal Flow targets; runner needs graph traversal")
        return non_terminal[0]
    if not flow_cases:
        raise AssertionError("Step has no Flow Route")
    return flow_cases[0]


def mock_value(
    condition_id: str,
    spec: dict[str, Any],
    step_id: str,
    scenario_id: str,
    requirement_root: pathlib.Path,
) -> Any:
    output_type = spec["type"]
    source = spec.get("source")
    if source:
        if source != "integration.trace.document_url" or output_type != "url":
            raise AssertionError(f"unsupported authoritative Output source {source!r}")
        return TRACE_DOCUMENT_URL
    if output_type == "enum_value":
        values = spec.get("values") or []
        if not values:
            raise AssertionError(f"{condition_id} enum_value has no values")
        return values[0]
    if output_type == "string":
        return f"e2e-agent-assertion:{step_id}:{condition_id}"
    if output_type == "boolean":
        return True
    if output_type == "integer":
        if spec.get("minimum") is not None:
            return spec["minimum"]
        maximum = spec.get("maximum")
        return maximum if maximum is not None and maximum < 0 else 0
    if output_type == "path":
        relative = pathlib.Path("mock") / f"{condition_id}.txt"
        path = requirement_root / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text(f"e2e-mock for {scenario_id}\n", encoding="utf-8")
        return relative.as_posix()
    if output_type == "url":
        token = hashlib.sha256(f"{scenario_id}:{condition_id}".encode()).hexdigest()[:20]
        return f"https://bytedance.larkoffice.com/docx/E2EMock{token}"
    if output_type == "url_list":
        count = max(1, spec.get("min_items") or 0)
        maximum = spec.get("max_items")
        if maximum is not None and count > maximum:
            raise AssertionError(f"{condition_id} url_list has incompatible item bounds")
        return [
            f"https://code.byted.org/e2e-mock/commonloop_cli/merge_requests/{7000 + index}"
            for index in range(count)
        ]
    if output_type == "object":
        return {"e2e_mock": True}
    raise AssertionError(f"unsupported Output type {output_type!r} for {condition_id}")


class Recorder:
    def __init__(self, binary: pathlib.Path, output_root: pathlib.Path):
        self.binary = binary
        self.output_root = output_root
        self.process_log = output_root / "process.log"
        self.process_log.write_text(
            "# Commonloop route-matrix Agent inputs, CLI responses and transitions\n\n",
            encoding="utf-8",
        )
        self.sequences: dict[str, int] = {}
        self.env = {
            key: value
            for key, value in os.environ.items()
            if key not in {"BOTMUX_CHAT_ID", "BOTMUX_SESSION_ID"}
        }

    def show(self, *lines: str) -> None:
        rendered = "\n".join(lines).rstrip() + "\n\n"
        sys.stdout.write(rendered)
        sys.stdout.flush()
        with self.process_log.open("a", encoding="utf-8") as handle:
            handle.write(rendered)

    def run(
        self,
        scenario_id: str,
        label: str,
        requirement_root: pathlib.Path,
        args: list[str],
        *,
        input_text: str | None = None,
        show: bool = False,
    ) -> subprocess.CompletedProcess[str]:
        sequence = self.sequences.get(scenario_id, 0) + 1
        self.sequences[scenario_id] = sequence
        audit = self.output_root / "audit" / scenario_id
        audit.mkdir(parents=True, exist_ok=True)
        prefix = audit / f"{sequence:03d}-{label}"
        argv = [str(self.binary), *args]
        started = now()
        start = time.monotonic()
        result = subprocess.run(
            argv,
            cwd=requirement_root,
            env=self.env,
            text=True,
            input=input_text,
            capture_output=True,
        )
        write_json(
            prefix.with_suffix(".command.json"),
            {
                "started_at": started,
                "finished_at": now(),
                "duration_seconds": round(time.monotonic() - start, 6),
                "cwd": str(requirement_root),
                "argv": argv,
                "exit_code": result.returncode,
            },
        )
        prefix.with_suffix(".stdout").write_text(result.stdout, encoding="utf-8")
        prefix.with_suffix(".stderr").write_text(result.stderr, encoding="utf-8")
        if input_text is not None:
            prefix.with_suffix(".stdin").write_text(input_text, encoding="utf-8")
        if show:
            block = [
                "$ " + shlex.join(argv),
                "",
                "[CLI -> Agent 原始响应]",
                result.stdout.rstrip(),
            ]
            if result.stderr:
                block.extend(["[stderr]", result.stderr.rstrip()])
            self.show(*block)
        return result


def success(result: subprocess.CompletedProcess[str], label: str) -> dict[str, Any]:
    if result.returncode != 0:
        detail = result.stderr.strip() or result.stdout.strip()
        raise AssertionError(f"{label} exited {result.returncode}: {detail}")
    try:
        value = json.loads(result.stdout)
    except json.JSONDecodeError as error:
        raise AssertionError(f"{label} returned non-JSON stdout: {error}") from error
    if value.get("ok") is not True:
        raise AssertionError(f"{label} returned ok={value.get('ok')!r}")
    return value


def status(recorder: Recorder, scenario_id: str, root: pathlib.Path, label: str) -> dict[str, Any]:
    return success(
        recorder.run(scenario_id, label, root, ["flow", "status", "--root", str(root)]),
        label,
    )


def state(root: pathlib.Path) -> dict[str, Any]:
    value = read_json(root / ".commonloop" / "flow" / "state.json")
    if "outputs" in value:
        raise AssertionError("Flow State still embeds Outputs")
    registry = read_json(root / ".commonloop" / "output" / "state.json")
    if registry["workflow"] != value["release"]["workflow"]:
        raise AssertionError("Output Registry Workflow differs from Flow State")
    value["outputs"] = registry["outputs"]
    return value


def events(root: pathlib.Path) -> list[dict[str, Any]]:
    path = root / ".commonloop" / "trace" / "events.jsonl"
    return [json.loads(line) for line in path.read_text(encoding="utf-8").splitlines() if line]


def condition_results(
    current_state: dict[str, Any],
    case: dict[str, Any],
    root: pathlib.Path,
) -> tuple[list[dict[str, Any]], list[str]]:
    available = {item["id"]: item for item in current_state["conditions"]}
    results, keys = [], []
    for condition_id in case["condition_ids"]:
        if condition_id not in available:
            raise AssertionError(f"Status omitted Condition {condition_id}")
        output = available[condition_id]["output"]
        keys.append(output["key"])
        results.append(
            {
                "condition_id": condition_id,
                "output": {
                    "type": output["type"],
                    "value": mock_value(
                        condition_id,
                        output,
                        case["source_step_id"],
                        case["scenario_id"],
                        root,
                    ),
                },
            }
        )
    return results, keys


def request_for_case(case: dict[str, Any], results: list[dict[str, Any]]) -> dict[str, Any]:
    summary = f"e2e-mock {case['scenario_id']}: {','.join(case['condition_ids'])}"
    return {
        "step_id": case["source_step_id"],
        "condition_results": results,
        "route": case["route"],
        "summary": summary,
        "evidence": [{"source": "ai", "content": summary, "ref": case["scenario_id"]}],
    }


def report_args(
    root: pathlib.Path,
    case: dict[str, Any],
    results: list[dict[str, Any]],
) -> tuple[list[str], dict[str, Any]]:
    request = request_for_case(case, results)
    evidence = request["evidence"][0]
    args = [
        "flow", "report", "result", "--root", str(root),
        "--step-id", request["step_id"],
    ]
    for result in request["condition_results"]:
        args.extend(
            [
                "--condition-result",
                json.dumps(result, ensure_ascii=False, separators=(",", ":")),
            ]
        )
    args.extend(
        [
            "--summary", request["summary"],
            "--evidence", json.dumps(evidence, ensure_ascii=False, separators=(",", ":")),
        ]
    )
    if "next_step_id" in request["route"]:
        args.extend(["--next-step-id", request["route"]["next_step_id"]])
    elif "back_step_id" in request["route"]:
        args.extend(["--back-step-id", request["route"]["back_step_id"]])
    else:
        args.append("--terminal")
    return args, request


def durable_bytes(root: pathlib.Path) -> tuple[bytes, ...]:
    return (
        (root / ".commonloop" / "flow" / "state.json").read_bytes(),
        (root / ".commonloop" / "output" / "state.json").read_bytes(),
        (root / ".commonloop" / "trace" / "events.jsonl").read_bytes(),
        (root / ".commonloop" / "card" / "projection.json").read_bytes(),
    )


def execute_rejected_case(
    recorder: Recorder,
    baseline: pathlib.Path,
    scenario_id: str,
    request: dict[str, Any],
    expected_code: str,
) -> dict[str, Any]:
    root = recorder.output_root / "negative" / scenario_id
    root.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(baseline, root)
    before_status = status(recorder, scenario_id, root, "status-before")
    before_files = durable_bytes(root)
    encoded = json.dumps(request, ensure_ascii=False)
    result = recorder.run(
        scenario_id,
        "flow-report-result",
        root,
        ["flow", "report", "result", "--root", str(root), "--input", "-"],
        input_text=encoded,
        show=True,
    )
    failures: list[str] = []
    observed_code = ""
    try:
        if result.returncode == 0:
            raise AssertionError("rejected request unexpectedly succeeded")
        envelope = json.loads(result.stderr)
        observed_code = envelope.get("error", {}).get("code", "")
        if observed_code != expected_code:
            raise AssertionError(f"error code {observed_code!r}, expected {expected_code!r}")
        after_status = status(recorder, scenario_id, root, "status-after")
        if before_status["data"]["state"] != after_status["data"]["state"]:
            raise AssertionError("rejected request changed projected State")
        if before_files != durable_bytes(root):
            raise AssertionError("rejected request changed State/Event files")
    except Exception as error:
        failures.append(str(error))
        audit = recorder.output_root / "audit" / scenario_id
        audit.mkdir(parents=True, exist_ok=True)
        (audit / "failure.traceback.txt").write_text(traceback.format_exc(), encoding="utf-8")
    recorder.show(
        f"[拒绝结果] {scenario_id}",
        f"Expected: {expected_code}",
        f"Observed: {observed_code or 'none'}",
        f"State/Event unchanged: {'PASS' if not failures else 'FAIL'}",
    )
    return {
        "scenario_id": scenario_id,
        "expected_code": expected_code,
        "observed_code": observed_code,
        "pass": not failures,
        "failures": failures,
        "requirement_root": str(root),
    }


def execute_dry_run(
    recorder: Recorder,
    baseline: pathlib.Path,
    case: dict[str, Any],
) -> dict[str, Any]:
    scenario_id = "dry-run-no-durable-effect"
    dry_root = recorder.output_root / "checks" / scenario_id / "dry"
    real_root = recorder.output_root / "checks" / scenario_id / "real"
    dry_root.parent.mkdir(parents=True, exist_ok=True)
    shutil.copytree(baseline, dry_root)
    shutil.copytree(baseline, real_root)
    failures: list[str] = []
    try:
        current_state = current(status(recorder, scenario_id, dry_root, "status-before"))
        if current_state is None:
            raise AssertionError("dry-run source has no current Step")
        results, _ = condition_results(current_state, case, dry_root)
        args, request = report_args(dry_root, case, results)
        before_files = durable_bytes(dry_root)
        dry = success(
            recorder.run(
                scenario_id,
                "dry-run",
                dry_root,
                [*args, "--dry-run"],
                show=True,
            ),
            "dry-run Result",
        )
        if "event_id" in dry["data"]:
            raise AssertionError("dry-run returned event_id")
        if before_files != durable_bytes(dry_root):
            raise AssertionError("dry-run changed State/Event files")
        real_args, _ = report_args(real_root, case, results)
        real = success(
            recorder.run(scenario_id, "real-result", real_root, real_args),
            "real Result",
        )
        real_data = dict(real["data"])
        real_data.pop("event_id", None)
        if dry["data"] != real_data:
            raise AssertionError("dry-run calculation differs from real Result")
        recorder.show(
            f"[Dry Run] {case['source_step_id']} -> {case['target_step_id']}",
            json.dumps(request, ensure_ascii=False, indent=2),
            "No Event / no State write: PASS",
        )
    except Exception as error:
        failures.append(str(error))
        audit = recorder.output_root / "audit" / scenario_id
        audit.mkdir(parents=True, exist_ok=True)
        (audit / "failure.traceback.txt").write_text(traceback.format_exc(), encoding="utf-8")
    return {"scenario_id": scenario_id, "pass": not failures, "failures": failures}


def verify_result(
    case: dict[str, Any],
    result: dict[str, Any],
    after: dict[str, Any],
    pre_state: dict[str, Any],
    output_keys: list[str],
    step_order: dict[str, int],
    root: pathlib.Path,
) -> dict[str, Any]:
    data = result["data"]
    transition = data["transition"]
    observed_target = transition.get("to_step_id") or "Done"
    expected = (
        data["effect"] == case["expected_effect"]
        and transition["direction"] == case["direction"]
        and transition["from_step_id"] == case["source_step_id"]
        and observed_target == case["target_step_id"]
    )
    if not expected:
        raise AssertionError(f"unexpected Result effect/transition: {data}")
    after_current = current(after)
    if case["expected_effect"] == "completed":
        if after["data"]["state"]["status"] != "completed" or after_current is not None:
            raise AssertionError("terminal Flow did not produce completed Status")
    elif after_current is None or after_current["context"]["step_id"] != case["target_step_id"]:
        raise AssertionError(f"post Status did not reach {case['target_step_id']}")
    if data["state"] != after["data"]["state"]:
        raise AssertionError("Result State differs from the immediately following Status State")

    durable_state = state(root)
    durable_events = events(root)
    if not durable_events:
        raise AssertionError("Event log is empty")
    event_id = data["event_id"]
    event_index = next(
        (index for index, item in enumerate(durable_events) if item["event_id"] == event_id),
        None,
    )
    if event_index is None:
        raise AssertionError("Result Event is missing from the durable log")
    event = durable_events[event_index]
    if event["kind"] != "flow_result":
        raise AssertionError("Result event_id does not identify a flow_result Event")
    previous_id = event_id
    for tail_event in durable_events[event_index + 1:]:
        if tail_event.get("caused_by_event_id") != previous_id:
            raise AssertionError("Events after Result do not form one causal chain")
        previous_id = tail_event["event_id"]
    if durable_state["last_event_id"] != durable_events[-1]["event_id"]:
        raise AssertionError("State last_event_id differs from the durable Event tail")
    payload = event["payload"]["flow_result"]
    if payload["effect"] != data["effect"] or payload["transition"] != transition:
        raise AssertionError("Result and Event effect/transition differ")
    if [item["condition_id"] for item in payload["condition_results"]] != case["condition_ids"]:
        raise AssertionError("Event ConditionResults differ from the submitted Route alternative")
    changes = payload.get("output_changes", {})
    if sorted(changes.get("accepted", [])) != sorted(set(output_keys)):
        raise AssertionError("Event accepted Output keys differ from submitted Conditions")

    post_outputs = durable_state.get("outputs", {})
    projection = read_json(root / ".commonloop" / "card" / "projection.json")
    if projection.get("outputs") != post_outputs:
        raise AssertionError("Card Projection Outputs differ from Output Registry")
    invalidated = sorted(changes.get("invalidated", []))
    if sorted(data["invalidated_outputs"]) != invalidated:
        raise AssertionError("Result invalidated_outputs differ from the durable Event")
    if case["direction"] == "flow":
        for key in output_keys:
            output = post_outputs.get(key)
            if output is None or output["producer_step_id"] != case["source_step_id"]:
                raise AssertionError(f"Flow Output {key} is missing or has the wrong producer")
    else:
        back_index = step_order[case["target_step_id"]]
        expected_invalidated = set(output_keys)
        for key, output in pre_state.get("outputs", {}).items():
            if step_order[output["producer_step_id"]] >= back_index:
                expected_invalidated.add(key)
        if invalidated != sorted(expected_invalidated):
            raise AssertionError(
                f"Loop invalidated {invalidated}, expected {sorted(expected_invalidated)}"
            )
        for key in expected_invalidated:
            if key in post_outputs:
                raise AssertionError(f"invalidated Output {key} remains in Output Registry")
        for key, output in pre_state.get("outputs", {}).items():
            if key not in expected_invalidated and post_outputs.get(key) != output:
                raise AssertionError(f"Loop changed preserved Output {key}")
    return {
        "observed_effect": data["effect"],
        "observed_target_step_id": observed_target,
        "event_id": event_id,
        "invalidated": invalidated,
    }


def diagnostics(recorder: Recorder, case: dict[str, Any], root: pathlib.Path) -> dict[str, Any]:
    trace = success(
        recorder.run(
            case["scenario_id"], "trace-render", root,
            ["trace", "render", "--root", str(root)],
        ),
        "trace render",
    )
    doctor = success(
        recorder.run(case["scenario_id"], "doctor", root, ["doctor", "--root", str(root)]),
        "doctor",
    )
    failed = [check["id"] for check in doctor["data"]["checks"] if check["status"] == "failed"]
    warnings = [check["id"] for check in doctor["data"]["checks"] if check["status"] == "warning"]
    if failed or sorted(set(warnings) - {"release_manifest"}):
        raise AssertionError(f"Doctor failed={failed}, warnings={warnings}")
    return {
        "trace_event_count": len(events(root)),
        "doctor_status": doctor["data"]["status"],
        "doctor_warnings": warnings,
        "trace_ok": trace["ok"],
    }


def execute_case(
    recorder: Recorder,
    case: dict[str, Any],
    root: pathlib.Path,
    step_order: dict[str, int],
) -> dict[str, Any]:
    failures: list[str] = []
    observed: dict[str, Any] = {}
    routed = False
    try:
        before = status(recorder, case["scenario_id"], root, "status-before")
        current_state = current(before)
        if current_state is None or current_state["context"]["step_id"] != case["source_step_id"]:
            raise AssertionError(f"scenario root is not at {case['source_step_id']}")
        matches = [item for item in route_cases(current_state) if item == case]
        if matches != [case]:
            raise AssertionError("Status does not expose the exact Route alternative")
        pre_state = state(root)
        results, output_keys = condition_results(current_state, case, root)
        args, request = report_args(root, case, results)
        recorder.show(
            f"=== {case['scenario_id']} ===",
            "[当前状态与目标]",
            f"Step: {case['source_step_id']}",
            f"Route: {case['direction']} -> {case['target_step_id']}",
            f"Conditions: {' + '.join(case['condition_ids'])}",
            "",
            "[模拟 Agent 参数]",
            json.dumps(request, ensure_ascii=False, indent=2),
            "",
            "[Agent -> CLI 命令]",
        )
        response = success(
            recorder.run(
                case["scenario_id"], "flow-report-result", root,
                args, show=True,
            ),
            "flow report result",
        )
        routed = True
        after = status(recorder, case["scenario_id"], root, "status-after")
        observed.update(
            verify_result(case, response, after, pre_state, output_keys, step_order, root)
        )
        next_state = current(after)
        recorder.show(
            "[推进结果]",
            f"Effect: {observed['observed_effect']}",
            (
                f"Transition: {case['source_step_id']} --{case['direction']}--> "
                f"{observed['observed_target_step_id']}"
            ),
            f"Current Step: {next_state['context']['step_id'] if next_state else 'Done'}",
            f"Accepted Outputs: {', '.join(output_keys) or 'none'}",
            f"Invalidated Outputs: {', '.join(observed['invalidated']) or 'none'}",
        )
        observed.update(diagnostics(recorder, case, root))
    except Exception as error:  # retain every failed scenario and its exact现场
        failures.append(str(error))
        audit = recorder.output_root / "audit" / case["scenario_id"]
        audit.mkdir(parents=True, exist_ok=True)
        (audit / "failure.traceback.txt").write_text(traceback.format_exc(), encoding="utf-8")
    return {
        **case,
        **observed,
        "requirement_root": str(root),
        "audit_dir": str(recorder.output_root / "audit" / case["scenario_id"]),
        "routed": routed,
        "pass": not failures,
        "failures": failures,
    }


def git(source: pathlib.Path, *args: str) -> str:
    result = subprocess.run(
        ["git", *args], cwd=source, text=True, capture_output=True, check=True
    )
    return result.stdout.rstrip("\n")


def render_report(matrix: dict[str, Any]) -> str:
    metadata, results, checks = matrix["metadata"], matrix["scenarios"], matrix["checks"]
    lines = [
        "# Commonloop 全路由矩阵 E2E",
        "",
        f"- 结果：**{metadata['passed']}/{metadata['discovered']} PASS**，失败 {metadata['failed']}。",
        f"- Flow：{metadata['flow_routes']} Route / {metadata['flow_alternatives']} alternatives。",
        f"- Loop：{metadata['loop_routes']} Route / {metadata['loop_alternatives']} alternatives。",
        f"- 负向与 dry-run：{sum(item['pass'] for item in checks)}/{len(checks)} PASS。",
        f"- Commit：`{metadata['source_commit']}`；Workflow：`{metadata.get('workflow_digest')}`。",
        f"- Source dirty：`{metadata['source_dirty']}`；执行前后不变：`{metadata['source_unchanged']}`；Binary SHA-256：`{metadata['binary_sha256']}`。",
        "- 每个场景均执行真实 CLI；mock 数据只验证路由机制，不代表业务事实。",
        "",
        "| 场景 | 条件 | 期望 | 实际 | 失效 Output | 结果 |",
        "|---|---|---|---|---|---|",
    ]
    for item in results:
        expected = f"{item['expected_effect']} → {item['target_step_id']}"
        actual = f"{item.get('observed_effect', '—')} → {item.get('observed_target_step_id', '—')}"
        invalidated = ", ".join(item.get("invalidated", [])) or "—"
        outcome = "PASS" if item["pass"] else "FAIL: " + "; ".join(item["failures"])
        lines.append(
            f"| `{item['scenario_id']}` | {' + '.join(item['condition_ids'])} | {expected} | "
            f"{actual} | {invalidated} | {outcome} |"
        )
    lines.extend(
        [
            "",
            "## 原子拒绝与 Dry Run",
            "",
            "| 场景 | 期望错误 | 实际错误 | 结果 |",
            "|---|---|---|---|",
            *[
                f"| `{item['scenario_id']}` | {item.get('expected_code', 'success without write')} | "
                f"{item.get('observed_code', '—')} | {'PASS' if item['pass'] else 'FAIL: ' + '; '.join(item['failures'])} |"
                for item in checks
            ],
            "",
            "## 过程文件",
            "",
            "- `process.log`：每条受测 `flow report result` 命令和原始响应。",
            "- `route-matrix.json`：完整机器结果与源码/二进制身份。",
            "- `audit/<scenario_id>/`：每条 CLI 的 argv、退出码、stdout、stderr 与失败 traceback。",
            "- `baseline/`、`cases/`：可直接用 `flow status` 复查的 Requirement 现场。",
            "",
        ]
    )
    return "\n".join(lines)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Run every Flow/Loop when.any_of alternative exposed by flow status."
    )
    parser.add_argument("--binary", required=True, help="source-built commonloop binary")
    parser.add_argument("--output-root", required=True, help="new directory for all audit files")
    parser.add_argument("--source-root", default=".", help="Git worktree used to build the binary")
    parser.add_argument("--workflow", default="technical-solution-design", help="Workflow id passed to flow init")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    source = pathlib.Path(args.source_root).resolve()
    binary = pathlib.Path(args.binary).resolve(strict=True)
    output_root = pathlib.Path(args.output_root).resolve()
    if output_root.exists():
        raise SystemExit(f"output root already exists: {output_root}")
    if not binary.is_file() or not os.access(binary, os.X_OK):
        raise SystemExit(f"binary is not executable: {binary}")
    initial_status = git(source, "status", "--short")
    source_commit = git(source, "rev-parse", "HEAD")
    source_branch = git(source, "branch", "--show-current")
    started_at = now()

    output_root.mkdir(parents=True)
    recorder = Recorder(binary, output_root)
    baseline = output_root / "baseline"
    baseline.mkdir()
    init = success(
        recorder.run(
            "setup", "flow-init", baseline,
            [
                "flow", "init", "--root", str(baseline), "--workflow", args.workflow,
                "--title", f"e2e-mock route matrix {source_commit[:12]}",
            ],
        ),
        "flow init",
    )
    workflow_digest = init["data"]["workflow"]["digest"]
    trace_bind = ["trace", "bind", "--root", str(baseline), "--document-url", TRACE_DOCUMENT_URL]
    if args.workflow == "commonloop-maintainer":
        trace_bind.extend(["--cli-log-document-url", CLI_LOG_DOCUMENT_URL])
    success(
        recorder.run(
            "setup", "trace-bind", baseline,
            trace_bind,
        ),
        "trace bind",
    )
    results: list[dict[str, Any]] = []
    checks: list[dict[str, Any]] = []
    visited: list[str] = []
    counts = {"flow_routes": 0, "flow_alternatives": 0, "loop_routes": 0, "loop_alternatives": 0}
    global_failures: list[str] = []
    negative_basics_done = False
    partial_match_done = False
    dry_run_done = False

    try:
        while True:
            baseline_status = status(recorder, "setup", baseline, "flow-status")
            current_state = current(baseline_status)
            if current_state is None:
                break
            step_id = current_state["context"]["step_id"]
            if step_id in visited:
                raise AssertionError(f"baseline Flow cycled at {step_id}")
            visited.append(step_id)
            step_order = {item: index for index, item in enumerate(visited)}
            cases = route_cases(current_state)
            flow_cases = [case for case in cases if case["direction"] == "flow"]
            if not flow_cases:
                raise AssertionError(f"{step_id} has no Flow Route")
            counts["flow_routes"] += sum(route["direction"] == "flow" for route in current_state["available_routes"])
            counts["loop_routes"] += sum(route["direction"] == "loop" for route in current_state["available_routes"])
            counts["flow_alternatives"] += len(flow_cases)
            counts["loop_alternatives"] += len(cases) - len(flow_cases)
            primary = select_baseline_flow_case(flow_cases)

            if not dry_run_done:
                checks.append(execute_dry_run(recorder, baseline, primary))
                dry_run_done = True

            if not negative_basics_done:
                loop_cases = [case for case in cases if case["direction"] == "loop"]
                if not loop_cases:
                    raise AssertionError(f"{step_id} has no Loop Route for negative checks")
                loop_case = loop_cases[0]
                flow_results, _ = condition_results(current_state, primary, baseline)
                loop_results, _ = condition_results(current_state, loop_case, baseline)
                flow_request = request_for_case(primary, flow_results)
                loop_request = request_for_case(loop_case, loop_results)
                unknown = dict(flow_request)
                unknown["route"] = {"next_step_id": "__missing_step__"}
                stale = dict(flow_request)
                stale["step_id"] = "__stale_step__"
                missing = dict(flow_request)
                missing.pop("route")
                multiple = dict(flow_request)
                multiple["route"] = {
                    "next_step_id": primary["route"].get("next_step_id", "__flow__"),
                    "back_step_id": loop_case["route"]["back_step_id"],
                }
                terminal_false = dict(flow_request)
                terminal_false["route"] = {"terminal": False}
                wrong_flow = dict(flow_request)
                wrong_flow["route"] = loop_case["route"]
                wrong_loop = dict(loop_request)
                wrong_loop["route"] = primary["route"]
                negative_requests = [
                    ("reject-missing-route", missing, "INVALID_ARGUMENT"),
                    ("reject-multiple-route-fields", multiple, "INVALID_ARGUMENT"),
                    ("reject-terminal-false", terminal_false, "INVALID_ARGUMENT"),
                    ("reject-unknown-target", unknown, "ROUTE_NOT_ALLOWED"),
                    ("reject-stale-step", stale, "STEP_NOT_CURRENT"),
                    ("reject-flow-facts-with-loop-route", wrong_flow, "ROUTE_NOT_MATCHED"),
                    ("reject-loop-facts-with-flow-route", wrong_loop, "ROUTE_NOT_MATCHED"),
                ]
                groups: dict[str, list[str]] = {}
                for condition in current_state["conditions"]:
                    group = condition.get("exclusive_group")
                    if group:
                        groups.setdefault(group, []).append(condition["id"])
                pair = next((ids[:2] for ids in groups.values() if len(ids) >= 2), None)
                if pair:
                    conflict_case = {**primary, "scenario_id": "reject-exclusive-conflict", "condition_ids": pair}
                    conflict_results, _ = condition_results(current_state, conflict_case, baseline)
                    conflict = request_for_case(conflict_case, conflict_results)
                    conflict["route"] = primary["route"]
                    negative_requests.append(
                        ("reject-exclusive-conflict", conflict, "CONDITION_CONFLICT")
                    )
                for scenario_id, request, expected_code in negative_requests:
                    checks.append(
                        execute_rejected_case(
                            recorder, baseline, scenario_id, request, expected_code
                        )
                    )
                negative_basics_done = True

            if not partial_match_done:
                complex_case = next(
                    (case for case in flow_cases if len(case["condition_ids"]) > 1),
                    None,
                )
                if complex_case:
                    partial_case = {
                        **complex_case,
                        "scenario_id": "reject-partial-condition-group",
                        "condition_ids": complex_case["condition_ids"][:1],
                    }
                    partial_results, _ = condition_results(current_state, partial_case, baseline)
                    checks.append(
                        execute_rejected_case(
                            recorder,
                            baseline,
                            partial_case["scenario_id"],
                            request_for_case(partial_case, partial_results),
                            "ROUTE_NOT_MATCHED",
                        )
                    )
                    partial_match_done = True

            for case in cases:
                if case == primary:
                    continue
                case_root = output_root / "cases" / case["scenario_id"]
                case_root.parent.mkdir(parents=True, exist_ok=True)
                shutil.copytree(baseline, case_root)
                results.append(execute_case(recorder, case, case_root, step_order))

            primary_result = execute_case(recorder, primary, baseline, step_order)
            results.append(primary_result)
            if not primary_result["routed"]:
                raise AssertionError(f"baseline could not leave {step_id}")
            if primary["expected_effect"] == "completed":
                break
    except Exception as error:
        global_failures.append(str(error))
        (output_root / "failure.traceback.txt").write_text(traceback.format_exc(), encoding="utf-8")

    final_status = git(source, "status", "--short")
    discovered = counts["flow_alternatives"] + counts["loop_alternatives"]
    failed = (
        sum(not item["pass"] for item in results)
        + sum(not item["pass"] for item in checks)
        + len(global_failures)
    )
    metadata = {
        "started_at": started_at,
        "finished_at": now(),
        "source_root": str(source),
        "source_branch": source_branch,
        "source_commit": source_commit,
        "source_dirty": initial_status != "",
        "source_unchanged": final_status == initial_status,
        "source_status_before": initial_status.splitlines(),
        "source_status_after": final_status.splitlines(),
        "binary": str(binary),
        "binary_sha256": hashlib.sha256(binary.read_bytes()).hexdigest(),
        "workflow_id": args.workflow,
        "workflow_digest": workflow_digest,
        **counts,
        "discovered": discovered,
        "executed": len(results),
        "passed": sum(item["pass"] for item in results),
        "failed": failed,
        "global_failures": global_failures,
    }
    matrix = {"schema_version": 1, "metadata": metadata, "scenarios": results, "checks": checks}
    write_json(output_root / "route-matrix.json", matrix)
    (output_root / "E2E_REPORT.md").write_text(render_report(matrix), encoding="utf-8")
    print(
        f"done: {metadata['passed']}/{discovered} passed; "
        f"process={recorder.process_log}; report={output_root / 'E2E_REPORT.md'}",
        flush=True,
    )
    complete = (
        len(results) == discovered
        and negative_basics_done
        and partial_match_done
        and dry_run_done
        and failed == 0
        and final_status == initial_status
    )
    return 0 if complete else 1


if __name__ == "__main__":
    sys.exit(main())
