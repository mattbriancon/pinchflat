#!/usr/bin/env python3
"""Removes t.Skip("BLOCKED: X unported") where X is now implemented, runs the
affected tests, and re-skips any that fail as NEEDS-FIX with the failure.

Usage: docs/go-port/unblock.py
"""
import os, re, subprocess, glob, sys

ROOT = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
CORE = os.path.join(ROOT, "internal", "core")
ENV = dict(os.environ, PATH="/usr/local/go/bin:" + os.environ["PATH"], GOTOOLCHAIN="auto")
SKIP = re.compile(r'^(\s*)t\.Skip\("BLOCKED: (\w+)[^"]*"\)\n', re.M)

def src():
    return "".join(open(f).read() for f in glob.glob(os.path.join(CORE, "*.go")) if not f.endswith("_test.go"))

def implemented(name, code):
    m = re.search(r'\nfunc (?:\([^)]*\) )?' + name + r'\(.*?\n}\n', code, re.S)
    return bool(m) and "unported" not in m.group(0)

def run(pattern):
    p = subprocess.run(["go", "test", "./internal/core/", "-count=1", "-run", pattern, "-v"],
                       cwd=ROOT, env=ENV, capture_output=True, text=True, timeout=900)
    return p.stdout + p.stderr

def test_funcs(path):
    return re.findall(r'^func (Test\w+)\(t \*testing\.T\)', open(path).read(), re.M)

code = src()
changed = {}
for path in glob.glob(os.path.join(CORE, "*_test.go")):
    s = open(path).read()
    new = SKIP.sub(lambda m: "" if implemented(m.group(2), code) else m.group(0), s)
    if new != s:
        open(path, "w").write(new)
        changed[path] = s
print(f"unskipped in {len(changed)} files")

for path, original in changed.items():
    funcs = test_funcs(path)
    out = run("^(" + "|".join(funcs) + ")$")
    if "[build failed]" in out or "[setup failed]" in out or "vet:" in out:
        open(path, "w").write(original)
        print(f"BUILD {os.path.basename(path)}: tree doesn't compile; restored skips")
        print("\n".join(l for l in out.splitlines() if ".go:" in l)[:2000])
        continue
    failing = set(re.findall(r'--- FAIL: \w+/(\S+)', out))
    panicked = re.search(r'panic: (.*)', out)
    if panicked:
        # a panic aborts the binary; find the running subtest
        running = re.findall(r'=== RUN\s+\w+/(\S+)', out)
        if running:
            failing.add(running[-1])
    if not failing and "FAIL" not in out.split("\n")[-2:][0]:
        print(f"ok   {os.path.basename(path)}")
        continue
    reason = (panicked.group(1)[:80] if panicked else "fails").replace('"', "'")
    s = open(path).read()
    for name in failing:
        go_name = name.replace("_", " ")
        # re-skip the t.Run whose name matches (spaces become _ in -v output)
        pat = re.compile(r'(t\.Run\("' + re.escape(go_name).replace("\\ ", "[ _]") + r'", func\(t \*testing\.T\) \{\n)')
        s = pat.sub(lambda m: m.group(1) + '\t\tt.Skip("NEEDS-FIX: ' + reason + '")\n', s, count=1)
    open(path, "w").write(s)
    print(f"fix  {os.path.basename(path)}: re-skipped {len(failing)} as NEEDS-FIX")
