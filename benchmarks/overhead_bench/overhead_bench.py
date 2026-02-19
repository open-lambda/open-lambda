#!/usr/bin/env python3
"""
Overhead Benchmark for Open Lambda

Measures the time spent inside lambda execution (Lambda Time) vs. framework
overhead (sandbox creation/unpause) by driving requests against a running
OL worker and collecting stats from the /stats endpoint.

Always performs a cold-start measurement first, then runs warm throughput
phases at each configured concurrency level.

Usage:
    python3 benchmarks/overhead_bench/overhead_bench.py
"""

import concurrent.futures
import json
import sys
import time
from urllib import request, error

# ---------------------------------------------------------------------------
# Configuration — edit these constants to change benchmark behaviour
# ---------------------------------------------------------------------------

HOST = "localhost"
PORT = "5000"
LAMBDA_NAME = "crunch"
REQUESTS_PER_PHASE = 100
CONCURRENCIES = [1, 10, 50]
WARMUP_REQUESTS = 5
LAMBDA_PAYLOAD = b'{"iterations": 500000}'

# ---------------------------------------------------------------------------
# Stat keys emitted by the OL worker (see common/stats.go, lambdaInstance.go,
# lambdaFunction.go, sockPool.go).  Not all may be present in every run.
# ---------------------------------------------------------------------------

STAT_KEYS = [
    "web-request",                              # total handler time
    "LambdaFunc.Invoke",                        # Invoke wrapper
    "LambdaInstance-WaitSandbox",               # sandbox acquire/create/unpause
    "LambdaInstance-WaitSandbox-Unpause",       # unpause subset
    "LambdaInstance-WaitSandbox-NoImportCache", # create from pool
    "LambdaInstance-ServeRequests",             # serve loop
    "LambdaInstance-RoundTrip",                 # actual HTTP to sandbox
    "Create()",                                 # full sandbox create
    "Create()/acquire-mem",                     # memory allocation step
    "Create()/acquire-cgroup",                  # cgroup allocation step
    "Create()/make-root-fs",                    # root filesystem setup
    "Create()/fork-proc",                       # fork from zygote
    "Create()/fresh-proc",                      # fresh process creation
]

# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def ol_url(path):
    return f"http://{HOST}:{PORT}{path}"


def invoke_lambda(payload=LAMBDA_PAYLOAD):
    """Invoke the lambda. Returns (success, elapsed_ms)."""
    url = ol_url(f"/run/{LAMBDA_NAME}")
    t0 = time.monotonic()
    try:
        req = request.Request(url, data=payload, method="POST",
                              headers={"Content-Type": "application/json"})
        with request.urlopen(req, timeout=120) as resp:
            resp.read()
            elapsed = (time.monotonic() - t0) * 1000
            return resp.status == 200, elapsed
    except error.HTTPError:
        elapsed = (time.monotonic() - t0) * 1000
        return False, elapsed
    except Exception:
        elapsed = (time.monotonic() - t0) * 1000
        return False, elapsed


def fetch_stats():
    """Fetch the /stats snapshot from the OL worker."""
    url = ol_url("/stats")
    try:
        req = request.Request(url)
        with request.urlopen(req, timeout=10) as resp:
            return json.loads(resp.read())
    except Exception as e:
        print(f"WARNING: could not fetch /stats: {e}", file=sys.stderr)
        return {}


def diff_stats(before, after):
    """Compute per-phase averages from two /stats snapshots.

    The /stats endpoint exposes .cnt, .ms-sum (raw cumulative), and .ms-avg.
    We diff the raw sums directly to get exact per-phase averages.

    Returns a dict with .cnt (delta count) and .ms-avg (phase average) for
    every stat name found.
    """
    diff = {}
    # Collect all stat base names
    names = set()
    for k in after:
        if k.endswith(".cnt"):
            names.add(k[:-4])

    for name in names:
        cnt_before = before.get(f"{name}.cnt", 0)
        cnt_after = after.get(f"{name}.cnt", 0)
        sum_before = before.get(f"{name}.ms-sum", 0)
        sum_after = after.get(f"{name}.ms-sum", 0)

        delta_cnt = cnt_after - cnt_before
        delta_sum = sum_after - sum_before

        diff[f"{name}.cnt"] = delta_cnt
        if delta_cnt > 0:
            diff[f"{name}.ms-avg"] = delta_sum / delta_cnt
        else:
            diff[f"{name}.ms-avg"] = 0

    return diff


def check_worker():
    """Verify the worker is reachable."""
    url = ol_url("/status")
    try:
        with request.urlopen(url, timeout=5) as resp:
            return resp.read().decode().strip() == "ready"
    except Exception:
        return False

# ---------------------------------------------------------------------------
# Benchmark phases
# ---------------------------------------------------------------------------

def run_phase(num_requests, concurrency):
    """
    Send *num_requests* at the given concurrency level.
    Returns (results, stats_diff) where results is a list of
    (success, elapsed_ms) tuples.
    """
    before = fetch_stats()

    results = []
    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
        futures = [pool.submit(invoke_lambda) for _ in range(num_requests)]
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())

    after = fetch_stats()
    return results, diff_stats(before, after)

# ---------------------------------------------------------------------------
# Analysis
# ---------------------------------------------------------------------------

def analyse_phase(label, results, sdiff, concurrency):
    """Return a summary dict for one benchmark phase."""
    successes = sum(1 for ok, _ in results if ok)
    failures = sum(1 for ok, _ in results if not ok)

    summary = {
        "label": label,
        "concurrency": concurrency,
        "total_requests": len(results),
        "successes": successes,
        "failures": failures,
    }

    # Server-side timing from /stats deltas (now proper per-phase averages)
    for key in STAT_KEYS:
        cnt = sdiff.get(f"{key}.cnt", 0)
        avg = sdiff.get(f"{key}.ms-avg", 0)
        summary[f"{key}.cnt"] = cnt
        summary[f"{key}.ms-avg"] = avg

    # Compute breakdown (server-side averages).
    # sandbox_wait and lambda_time are sequential, not nested.
    #   total        = web-request                (entire handler lifetime)
    #   lambda_time  = LambdaInstance-RoundTrip   (lambda execution)
    #   sandbox_wait = LambdaInstance-WaitSandbox (create/unpause sandbox)
    web_avg = sdiff.get("web-request.ms-avg", 0)
    rt_avg = sdiff.get("LambdaInstance-RoundTrip.ms-avg", 0)
    sandbox_avg = sdiff.get("LambdaInstance-WaitSandbox.ms-avg", 0)

    summary["server_total_avg_ms"] = web_avg
    summary["lambda_time_avg_ms"] = rt_avg
    summary["sandbox_wait_avg_ms"] = sandbox_avg

    return summary

# ---------------------------------------------------------------------------
# Main
# ---------------------------------------------------------------------------

def main():
    # Verify worker is up
    print(f"Checking worker at {HOST}:{PORT} ...")
    if not check_worker():
        print("ERROR: worker is not reachable. Start it first with: sudo -A ./ol worker",
              file=sys.stderr)
        sys.exit(1)
    print("Worker is ready.\n")

    all_summaries = []

    # --- Cold-start phase (no warmup, single request) ---
    print("=" * 60)
    print("Phase: COLD START (concurrency=1, 1 request)")
    print("=" * 60)
    results, sdiff = run_phase(1, 1)
    summary = analyse_phase("cold-start", results, sdiff, 1)
    all_summaries.append(summary)
    print(f"  total:            {summary['server_total_avg_ms']:.2f}ms")
    print(f"  lambda time:      {summary['lambda_time_avg_ms']:.2f}ms")
    print(f"  sandbox wait:     {summary['sandbox_wait_avg_ms']:.2f}ms")
    print()

    # --- Warmup (prime the sandbox so subsequent phases are warm) ---
    if WARMUP_REQUESTS > 0:
        print(f"Warming up with {WARMUP_REQUESTS} sequential requests ...")
        for i in range(WARMUP_REQUESTS):
            ok, elapsed = invoke_lambda()
            status = "OK" if ok else "FAIL"
            print(f"  warmup {i+1}/{WARMUP_REQUESTS}: {elapsed:.1f}ms [{status}]")
        print()

    # --- Warm throughput phases at each concurrency level ---
    for c in CONCURRENCIES:
        print("=" * 60)
        print(f"Phase: concurrency={c}, requests={REQUESTS_PER_PHASE}")
        print("=" * 60)

        results, sdiff = run_phase(REQUESTS_PER_PHASE, c)
        label = f"c={c}"
        summary = analyse_phase(label, results, sdiff, c)
        all_summaries.append(summary)

        print(f"  results: {summary['successes']} ok, {summary['failures']} failed")
        print(f"  total={summary['server_total_avg_ms']:.1f}ms  "
              f"lambda time={summary['lambda_time_avg_ms']:.1f}ms  "
              f"sandbox wait={summary['sandbox_wait_avg_ms']:.1f}ms")
        print()

    # --- Print final summary table ---
    print("=" * 80)
    print(f"{'Phase':<14} {'Reqs':>5} {'OK':>5} {'Fail':>5} "
          f"{'Total':>8} {'Lambda':>10} {'SandboxWait':>12} {'SW %':>6}")
    print("-" * 80)
    for s in all_summaries:
        total = s["server_total_avg_ms"]
        lt = s["lambda_time_avg_ms"]
        sw = s["sandbox_wait_avg_ms"]
        combined = lt + sw
        sw_pct = (sw / combined * 100) if combined > 0 else 0
        print(f"{s['label']:<14} {s['total_requests']:>5} {s['successes']:>5} "
              f"{s['failures']:>5} "
              f"{total:>6.1f}ms {lt:>8.1f}ms {sw:>10.1f}ms {sw_pct:>5.1f}%")
    print("=" * 80)


if __name__ == "__main__":
    main()
