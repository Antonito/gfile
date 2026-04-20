#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""End-to-end benchmark for `gfile bench`.

Spawns two `go run . bench ...` processes per run, pipes SDPs between them,
parses the final upload/download stats, and aggregates across runs.
"""

from __future__ import annotations

import argparse
import json
import statistics
import subprocess
import sys
import threading
import time


def summarize(values: list[float]) -> dict[str, float | None]:
    """Return {min, median, mean, stdev, max} over a non-empty list.

    stdev is sample stdev; returns None when len(values) < 2.
    """
    if not values:
        raise ValueError("summarize requires at least one value")
    return {
        "min": min(values),
        "median": statistics.median(values),
        "mean": statistics.fmean(values),
        "stdev": statistics.stdev(values) if len(values) > 1 else None,
        "max": max(values),
    }


import os

GO_CMD = (
    [os.environ["GFILE_BIN"]]
    if os.environ.get("GFILE_BIN")
    else ["go", "run", "."]
)


class BenchRunError(RuntimeError):
    """Raised when a single bench run fails."""

    def __init__(self, message: str, *, stderr: str = "") -> None:
        super().__init__(message)
        self.stderr = stderr


def _read_events(proc, deadline):
    """Yield parsed JSON events from proc.stdout until EOF or timeout.

    Lines must be NDJSON under `gfile --json-output`. Anything non-JSON is
    a protocol violation and raises BenchRunError immediately — text
    prompts or progress lines would indicate the flag was dropped.
    """
    assert proc.stdout is not None
    for line in proc.stdout:
        if time.monotonic() > deadline:
            raise BenchRunError("timed out reading events")
        line = line.strip()
        if not line:
            continue
        try:
            yield json.loads(line)
        except json.JSONDecodeError as exc:
            raise BenchRunError(f"non-JSON on stdout: {line!r}") from exc


def _drain_stream(stream, out_list: list[str]) -> None:
    """Read every line from stderr into out_list until EOF.

    Called on child stderr so bench diagnostics and zerolog output don't
    fill the kernel pipe buffer and block the child. Run in a daemon
    thread; the child exits naturally and this returns at EOF.
    """
    try:
        for line in iter(stream.readline, ""):
            out_list.append(line)
    finally:
        stream.close()


class _RunProgress:
    """Thread-safe per-run progress state shared between drain and monitor.

    Also collects sender-side bandwidth samples (one per progress event) so
    the run can report a windowed peak/p50/min alongside the sustained rate.
    """

    def __init__(self) -> None:
        self._lock = threading.Lock()
        self.sender_bw = 0.0
        self.sender_bytes = 0  # bytes sent by the sender so far
        self.total_bytes = 0  # total bytes to transfer; 0 = unknown yet
        self.start_time = time.monotonic()
        # Sender bandwidth samples in MiB/s. The emitter fires one progress
        # event before any bytes flow (see pkg/transfer/progress.go), which
        # reports bw=0; we skip those so the distribution reflects real
        # transfer rate only.
        self.sender_bw_samples: list[float] = []

    def update(self, role: str, verb: str, bw: float, sent_bytes: int | None) -> None:
        # Only the sender's upload rate is meaningful for live display;
        # the receiver's rate on the same bytes is redundant.
        if role == "sender":
            with self._lock:
                self.sender_bw = bw
                if sent_bytes is not None:
                    self.sender_bytes = sent_bytes
                if bw > 0:
                    self.sender_bw_samples.append(bw)

    def set_total_bytes(self, n: int) -> None:
        with self._lock:
            if self.total_bytes == 0:
                self.total_bytes = n

    def snapshot(self) -> dict:
        with self._lock:
            return {
                "sender_bw": self.sender_bw,
                "sender_bytes": self.sender_bytes,
                "total_bytes": self.total_bytes,
                "elapsed": time.monotonic() - self.start_time,
            }

    def bw_distribution(self) -> dict[str, float | int] | None:
        """Return peak/p50/min over collected sender samples, or None if empty."""
        with self._lock:
            samples = list(self.sender_bw_samples)
        if not samples:
            return None
        return {
            "peak_mib_per_sec": max(samples),
            "p50_mib_per_sec": statistics.median(samples),
            "min_mib_per_sec": min(samples),
            "samples": len(samples),
        }


def _render_bar(done: float, total: float, width: int = 20) -> str:
    if total <= 0:
        return "…" * width
    frac = done / total
    if frac > 1.0:
        frac = 1.0
    filled = int(frac * width)
    return "█" * filled + "░" * (width - filled)


_MIB = 1024 * 1024


def _monitor_progress(
    progress: _RunProgress,
    stop_event: threading.Event,
    prefix: str,
    out=sys.stderr,
) -> None:
    """Re-render progress line on `out` until stop_event is set.

    Progress is size-based: sent/total bytes, rendered as MiB. If the child
    hasn't reported a total yet, the bar stays at indeterminate (…) and the
    line just shows bandwidth.
    """
    last_len = 0
    try:
        while not stop_event.wait(0.25):
            snap = progress.snapshot()
            sent = snap["sender_bytes"]
            total = snap["total_bytes"]
            bar = _render_bar(sent, total)
            sent_mib = sent / _MIB
            if total > 0:
                total_mib = total / _MIB
                suffix = f"{sent_mib:6.1f}/{total_mib:.1f} MiB"
            else:
                suffix = f"{sent_mib:6.1f} MiB"
            line = (
                f"{prefix} {snap['sender_bw']:6.2f} MiB/s  "
                f"[{bar}] {suffix}"
            )
            padded = line + " " * max(0, last_len - len(line))
            print(f"\r{padded}", end="", file=out, flush=True)
            last_len = len(line)
    finally:
        # Clear the rendered line
        print("\r" + " " * last_len + "\r", end="", file=out, flush=True)


def _consume_events(proc, role, progress, stats_slot, deadline):
    """Drain events from one side. Updates progress state and fills stats_slot[0]."""
    try:
        for ev in _read_events(proc, deadline):
            t = ev.get("type")
            if t == "progress":
                if progress is not None:
                    progress.update(
                        role,
                        "up" if role == "sender" else "dn",
                        float(ev.get("bytes_per_sec", 0.0)) / (1024 * 1024),
                        int(ev.get("bytes", 0)),
                    )
            elif t == "stats":
                stats_slot[0] = ev
            elif t == "error":
                raise BenchRunError(f"{role}: {ev.get('message', '<no message>')} ({ev.get('kind', '?')})")
            # `sdp` events were consumed before this loop starts; any
            # later `sdp` event would be a bug — ignore silently.
    finally:
        if proc.stdout is not None:
            proc.stdout.close()


def run_one(
    timeout: float = 120.0,
    progress: _RunProgress | None = None,
    size_mb: int | None = None,
    connections: int | None = None,
) -> dict[str, float | int | dict[str, float | int] | None]:
    deadline = time.monotonic() + timeout

    sender_args = [*GO_CMD, "--json-output", "bench", "-s", "--loopback"]
    receiver_args = [*GO_CMD, "--json-output", "bench", "--loopback"]
    if size_mb is not None:
        sender_args += ["--size", str(size_mb)]
    if connections is not None:
        sender_args += ["--connections", str(connections)]
        receiver_args += ["--connections", str(connections)]

    sender = subprocess.Popen(
        sender_args,
        stdin=subprocess.PIPE,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        bufsize=1,
    )
    sender_err_buf: list[str] = []
    receiver_err_buf: list[str] = []

    sender_err_thread = threading.Thread(
        target=_drain_stream, args=(sender.stderr, sender_err_buf), daemon=True,
    )
    sender_err_thread.start()

    receiver: subprocess.Popen[str] | None = None
    receiver_err_thread: threading.Thread | None = None
    sender_consumer: threading.Thread | None = None
    receiver_consumer: threading.Thread | None = None
    sender_stats: list[dict | None] = [None]
    receiver_stats: list[dict | None] = [None]
    try:
        # Sender emits `bench_total` before `sdp` (see cmd/bench.go:
        # BenchTotal runs before sess.Start). Capture the total here so
        # the live progress bar can render a fill ratio; stop as soon
        # as we see the sdp event.
        sender_sdp_ev: dict | None = None
        for ev in _read_events(sender, deadline):
            t = ev.get("type")
            if t == "bench_total" and progress is not None:
                progress.set_total_bytes(int(ev.get("bytes", 0)))
            elif t == "sdp":
                sender_sdp_ev = ev
                break
        if sender_sdp_ev is None:
            raise BenchRunError("sender closed stdout before emitting sdp")

        receiver = subprocess.Popen(
            receiver_args,
            stdin=subprocess.PIPE,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            text=True,
            bufsize=1,
        )
        receiver_err_thread = threading.Thread(
            target=_drain_stream, args=(receiver.stderr, receiver_err_buf), daemon=True,
        )
        receiver_err_thread.start()

        assert receiver.stdin is not None
        receiver.stdin.write(sender_sdp_ev["sdp"] + "\n")
        receiver.stdin.flush()

        receiver_sdp_ev = next(
            (ev for ev in _read_events(receiver, deadline) if ev.get("type") == "sdp"),
            None,
        )
        if receiver_sdp_ev is None:
            raise BenchRunError("receiver closed stdout before emitting sdp")
        assert sender.stdin is not None
        sender.stdin.write(receiver_sdp_ev["sdp"] + "\n")
        sender.stdin.flush()

        # Consume remaining events from both sides concurrently.
        sender_consumer = threading.Thread(
            target=_consume_events,
            args=(sender, "sender", progress, sender_stats, deadline),
            daemon=True,
        )
        receiver_consumer = threading.Thread(
            target=_consume_events,
            args=(receiver, "receiver", progress, receiver_stats, deadline),
            daemon=True,
        )
        sender_consumer.start()
        receiver_consumer.start()

        remaining = max(deadline - time.monotonic(), 1.0)
        try:
            sender.wait(timeout=remaining)
        except subprocess.TimeoutExpired as exc:
            raise BenchRunError("sender did not exit within timeout") from exc
        remaining = max(deadline - time.monotonic(), 1.0)
        try:
            receiver.wait(timeout=remaining)
        except subprocess.TimeoutExpired as exc:
            raise BenchRunError("receiver did not exit within timeout") from exc

        sender_consumer.join(timeout=5)
        receiver_consumer.join(timeout=5)
        sender_err_thread.join(timeout=5)
        receiver_err_thread.join(timeout=5)

        sender_err = "".join(sender_err_buf)
        receiver_err = "".join(receiver_err_buf)

        if sender.returncode != 0:
            raise BenchRunError(
                f"sender exited with code {sender.returncode}",
                stderr=sender_err,
            )
        if receiver.returncode != 0:
            raise BenchRunError(
                f"receiver exited with code {receiver.returncode}",
                stderr=receiver_err,
            )

        if sender_stats[0] is None:
            raise BenchRunError("sender produced no stats event", stderr=sender_err)
        if receiver_stats[0] is None:
            raise BenchRunError("receiver produced no stats event", stderr=receiver_err)

        s_bytes = int(sender_stats[0]["bytes"])
        r_bytes = int(receiver_stats[0]["bytes"])
        if s_bytes != r_bytes:
            print(
                f"byte mismatch: sender sent {s_bytes}, receiver got {r_bytes}",
                file=sys.stderr,
            )
        bps = float(sender_stats[0]["bytes_per_sec"])
        result: dict[str, float | int | dict[str, float | int] | None] = {
            "bytes": s_bytes,
            "mib_per_sec": bps / (1024 * 1024),
            "receiver_bytes": r_bytes,
            "windowed": progress.bw_distribution() if progress is not None else None,
        }
        return result
    finally:
        for p in (sender, receiver):
            if p is not None and p.poll() is None:
                p.kill()
                p.wait(timeout=5)


def _to_bps(mib_per_sec: float) -> float:
    return mib_per_sec * _MIB


def format_human(runs: list[dict]) -> str:
    n = len(runs)
    lines = []
    for i, r in enumerate(runs, 1):
        lines.append(
            f"run {i}/{n}: {r['mib_per_sec']:.2f} MiB/s sustained  ({r['bytes']} B)"
        )
        windowed = r.get("windowed")
        if windowed is not None:
            lines.append(
                f"         windowed: peak {windowed['peak_mib_per_sec']:.2f}  "
                f"p50 {windowed['p50_mib_per_sec']:.2f}  "
                f"min {windowed['min_mib_per_sec']:.2f} MiB/s "
                f"(n={windowed['samples']})"
            )
    summary = summarize([float(r["mib_per_sec"]) for r in runs])
    lines.append("")
    lines.append(f"Summary ({n} run{'s' if n != 1 else ''}):")
    lines.append(f"  Sustained: {_fmt_summary(summary)} MiB/s")
    # Across-runs peak is useful to eyeball "did any run manage to top out higher?"
    peaks = [
        float(r["windowed"]["peak_mib_per_sec"])
        for r in runs
        if r.get("windowed") is not None
    ]
    if peaks:
        lines.append(f"  Peak (max across runs): {max(peaks):.2f} MiB/s")
    return "\n".join(lines)


def _fmt_summary(s: dict[str, float | None]) -> str:
    stdev = f"{s['stdev']:.2f}" if s["stdev"] is not None else "n/a"
    return (
        f"min {s['min']:.2f}  median {s['median']:.2f}  "
        f"mean {s['mean']:.2f}  stdev {stdev}  max {s['max']:.2f}"
    )


def format_json(runs: list[dict]) -> str:
    out_runs = []
    for r in runs:
        entry: dict[str, object] = {
            "bytes_per_sec": _to_bps(float(r["mib_per_sec"])),
            "bytes": int(r["bytes"]),
        }
        windowed = r.get("windowed")
        if windowed is not None:
            entry["windowed"] = {
                "peak_bytes_per_sec": _to_bps(float(windowed["peak_mib_per_sec"])),
                "p50_bytes_per_sec": _to_bps(float(windowed["p50_mib_per_sec"])),
                "min_bytes_per_sec": _to_bps(float(windowed["min_mib_per_sec"])),
                "samples": int(windowed["samples"]),
            }
        out_runs.append(entry)
    bps = [float(r["bytes_per_sec"]) for r in out_runs]
    return json.dumps(
        {
            "runs": out_runs,
            "summary": {"bytes_per_sec": summarize(bps)},
        },
        indent=2,
    )


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(
        description="Run gfile bench end-to-end and aggregate bandwidth.",
    )
    parser.add_argument(
        "-n", "--runs", type=int, default=1,
        help="Number of benchmark runs (default: 1).",
    )
    parser.add_argument(
        "--json", action="store_true",
        help="Emit a single JSON object on stdout instead of a human table.",
    )
    parser.add_argument(
        "--timeout", type=float, default=120.0,
        help="Per-run timeout in seconds (default: 120).",
    )
    parser.add_argument(
        "--size", type=int, default=None,
        help="Pass --size N (MB) through to gfile bench sender (default: gfile's default).",
    )
    parser.add_argument(
        "--connections", type=int, default=None,
        help="Pass --connections N through to both sides (default: gfile's default = 1).",
    )
    args = parser.parse_args(argv)

    if args.runs < 1:
        parser.error("--runs must be >= 1")

    progress_stream = sys.stderr if args.json else sys.stdout
    interactive = not args.json
    results: list[dict] = []
    for i in range(1, args.runs + 1):
        print(f"[{i}/{args.runs}] running bench...", file=progress_stream, flush=True)
        # Always collect samples — cheap, and the windowed distribution is
        # useful in both human and JSON output. Only the live render thread
        # is interactive-only.
        run_progress = _RunProgress()
        monitor_stop: threading.Event | None = None
        monitor_thread: threading.Thread | None = None
        if interactive:
            monitor_stop = threading.Event()
            monitor_thread = threading.Thread(
                target=_monitor_progress,
                args=(run_progress, monitor_stop, f"[{i}/{args.runs}]"),
                daemon=True,
            )
            monitor_thread.start()
        try:
            result = run_one(
                timeout=args.timeout,
                progress=run_progress,
                size_mb=args.size,
                connections=args.connections,
            )
            error: BenchRunError | None = None
        except BenchRunError as exc:
            error = exc
            result = None  # type: ignore[assignment]

        # Stop and clear the progress line before printing anything else.
        if monitor_stop is not None:
            monitor_stop.set()
        if monitor_thread is not None:
            monitor_thread.join(timeout=2)

        if error is not None:
            print(f"run {i} failed: {error}", file=sys.stderr)
            if error.stderr:
                print("--- child stderr ---", file=sys.stderr)
                print(error.stderr, file=sys.stderr)
            return 1

        results.append(result)
        print(
            f"[{i}/{args.runs}] {result['mib_per_sec']:.2f} MiB/s",
            file=progress_stream,
            flush=True,
        )

    if args.json:
        print(format_json(results))
    else:
        print()
        print(format_human(results))
    return 0


if __name__ == "__main__":
    sys.exit(main())
