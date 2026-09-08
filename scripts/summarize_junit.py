#!/usr/bin/env python3
"""Summarize a JUnit XML test report (e.g. from `gotestsum --junitfile`).

Reads structured testsuite/testcase data — never pretty-printed console
output — and prints surefire-style counts plus the failed-test list.

Usage:
    summarize_junit.py <report.xml>

Exit codes:
    0 — no failures or errors.
    1 — one or more failures/errors (capped reporting, still lists all).
    2 — report missing or unparseable (never prints a green summary).
"""

import sys
import xml.etree.ElementTree as ET


def parse_report(path):
    """Return (suites, problems) where problems is a list of failed/errored test ids."""
    try:
        tree = ET.parse(path)
    except FileNotFoundError:
        return None, [f"report not found: {path}"]
    except ET.ParseError as exc:
        return None, [f"could not parse {path}: {exc}"]

    root = tree.getroot()
    if root.tag == "testsuites":
        suites = list(root.iter("testsuite"))
    elif root.tag == "testsuite":
        suites = [root]
    else:
        return None, [f"unexpected root element <{root.tag}> in {path}"]

    totals = {"tests": 0, "failures": 0, "errors": 0, "skipped": 0}
    problems = []
    for suite in suites:
        for key in totals:
            try:
                totals[key] += int(suite.get(key, 0))
            except ValueError:
                pass
        suite_name = suite.get("name", "?")
        for case in suite.iter("testcase"):
            test_id = f"{case.get('classname', suite_name)}.{case.get('name', '?')}"
            for tag in ("failure", "error"):
                if case.find(tag) is not None:
                    problems.append(test_id)
                    break

    passed = totals["tests"] - totals["failures"] - totals["errors"] - totals["skipped"]
    return {"passed": max(passed, 0), **totals}, problems


def main(argv):
    if len(argv) != 2:
        print(f"usage: {argv[0]} <report.xml>", file=sys.stderr)
        return 2

    summary, problems = parse_report(argv[1])
    if summary is None:
        for problem in problems:
            print(f"❌ {problem}", file=sys.stderr)
        return 2

    print("")
    print("==========================================")
    print("           📊 TEST SUMMARY (JUnit)")
    print("==========================================")
    print("")
    print(f"📈 Total:   {summary['tests']}")
    print(f"✅ Passed:  {summary['passed']}")
    print(f"❌ Failed:  {summary['failures']}")
    if summary["errors"]:
        print(f"💥 Errors:  {summary['errors']}")
    print(f"⏭️  Skipped: {summary['skipped']}")
    print("")

    bad = summary["failures"] + summary["errors"]
    if bad > 0:
        print("==========================================")
        print("           ❌ FAILED TESTS")
        print("==========================================")
        for test_id in sorted(set(problems)):
            print(test_id)
        return 1

    print("🎉 All tests passed!")
    return 0


if __name__ == "__main__":
    sys.exit(main(sys.argv))
