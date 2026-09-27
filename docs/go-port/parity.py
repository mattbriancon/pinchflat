#!/usr/bin/env python3
"""Checks every Elixir test has a same-named Go t.Run in the mapped test file.

Usage: docs/go-port/parity.py [substring-filter]
Exit status 1 if any mapped test is missing.
"""
import os, re, sys

ROOT = os.path.join(os.path.dirname(__file__), "..", "..")
MANIFEST = os.path.join(ROOT, "docs", "go-port", "MANIFEST.md")
ELIXIR_TEST = re.compile(r'^\s*test\s+"((?:[^"\\]|\\.)*)"', re.M)
GO_RUN = re.compile(r't\.Run\(\s*"((?:[^"\\]|\\.)*)"\s*,')
ASSERTION = re.compile(r't\.(Error|Errorf|Fatal|Fatalf|Fail|FailNow|Skip)\b|Assert|Refute|coretest\.Assert')

def go_runs(src):
    """Names of t.Run blocks that contain at least one assertion (or an
    explicit skip). Empty or assertion-free bodies don't count as ported."""
    names = set()
    for m in GO_RUN.finditer(src):
        # body = text from this t.Run to the next t.Run / func at col 0
        start = m.end()
        nxt = GO_RUN.search(src, start)
        end = nxt.start() if nxt else len(src)
        top = src.find("\nfunc ", start)
        if top != -1 and top < end:
            end = top
        if ASSERTION.search(src[start:end]):
            names.add(m.group(1))
    return names

def rows():
    for line in open(MANIFEST):
        cells = [c.strip().strip("`") for c in line.split("|")[1:-1]]
        if len(cells) == 5 and cells[3].startswith("test/"):
            yield cells[1], cells[2], cells[3]

def go_test_path(target):
    return os.path.join(ROOT, target[:-len(os.path.splitext(target)[1])] + "_test.go")

def main():
    flt = sys.argv[1] if len(sys.argv) > 1 else ""
    missing_total = total = ported = 0
    for src, target, test in rows():
        if flt and flt not in src and flt not in target:
            continue
        ex_names = ELIXIR_TEST.findall(open(os.path.join(ROOT, test)).read())
        total += len(ex_names)
        gp = go_test_path(target)
        go_names = go_runs(open(gp).read()) if os.path.exists(gp) else set()
        missing = [n for n in ex_names if n not in go_names]
        ported += len(ex_names) - len(missing)
        status = "ok " if not missing else ("-- " if not go_names else "!! ")
        print(f"{status}{len(ex_names)-len(missing):4}/{len(ex_names):<4} {test}")
        if missing and go_names:
            for n in missing:
                print(f"        missing: {n}")
        missing_total += len(missing)
    print(f"\n{ported}/{total} Elixir tests have a Go t.Run with the same name")
    sys.exit(1 if missing_total else 0)

main()
