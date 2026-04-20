#!/usr/bin/env -S uv run --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
# ///
"""End-to-end CLI smoke test for `gfile`.

Spawns `gfile send` and `gfile receive` as subprocesses, pipes the NDJSON
SDP events between them via stdin/stdout, waits for both to exit, and
verifies the received file matches the fixture byte-for-byte.

Defaults to `go run .`; set GFILE_BIN=<path> to exercise a prebuilt
binary.
"""

from __future__ import annotations

import hashlib
import json
import os
import subprocess
import sys
import tempfile
import threading
import time
from pathlib import Path

REPO_ROOT = Path(__file__).resolve().parent.parent
FIXTURE = REPO_ROOT / "testdata" / "sample.bin"

GFILE_CMD = (
    [os.environ["GFILE_BIN"]]
    if os.environ.get("GFILE_BIN")
    else ["go", "run", "."]
)

TIMEOUT_SEC = 30.0


class E2EError(RuntimeError):
    pass


def _drain_stream(stream, out_list: list[str]) -> None:
    try:
        for line in iter(stream.readline, ""):
            out_list.append(line)
    finally:
        stream.close()


def _read_sdp(proc: subprocess.Popen, role: str, deadline: float) -> str:
    """Read NDJSON events from proc.stdout until an sdp event arrives."""
    assert proc.stdout is not None
    for line in proc.stdout:
        if time.monotonic() > deadline:
            raise E2EError(f"{role}: timed out waiting for sdp event")
        line = line.strip()
        if not line:
            continue
        try:
            ev = json.loads(line)
        except json.JSONDecodeError as exc:
            raise E2EError(f"{role}: non-JSON on stdout: {line!r}") from exc
        if ev.get("type") == "sdp":
            sdp = ev.get("sdp")
            if not isinstance(sdp, str) or not sdp:
                raise E2EError(f"{role}: sdp event missing 'sdp' field: {ev!r}")
            return sdp
    raise E2EError(f"{role}: stdout closed before emitting sdp event")


def _sha256(path: Path) -> str:
    h = hashlib.sha256()
    with open(path, "rb") as f:
        for chunk in iter(lambda: f.read(65536), b""):
            h.update(chunk)
    return h.hexdigest()


def run() -> int:
    if not FIXTURE.is_file():
        print(f"fixture not found: {FIXTURE}", file=sys.stderr)
        return 1

    deadline = time.monotonic() + TIMEOUT_SEC
    with tempfile.TemporaryDirectory(prefix="gfile-e2e-") as tmp:
        out_path = Path(tmp) / "received.bin"

        sender_args = [
            *GFILE_CMD, "--json-output", "send",
            "-f", str(FIXTURE),
        ]
        receiver_args = [
            *GFILE_CMD, "--json-output", "receive",
            "-o", str(out_path),
        ]

        sender = subprocess.Popen(
            sender_args, cwd=REPO_ROOT,
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
            text=True, bufsize=1,
        )
        sender_err: list[str] = []
        threading.Thread(
            target=_drain_stream, args=(sender.stderr, sender_err), daemon=True,
        ).start()

        receiver: subprocess.Popen | None = None
        receiver_err: list[str] = []
        try:
            sender_sdp = _read_sdp(sender, "sender", deadline)

            receiver = subprocess.Popen(
                receiver_args, cwd=REPO_ROOT,
                stdin=subprocess.PIPE, stdout=subprocess.PIPE, stderr=subprocess.PIPE,
                text=True, bufsize=1,
            )
            threading.Thread(
                target=_drain_stream, args=(receiver.stderr, receiver_err), daemon=True,
            ).start()

            assert receiver.stdin is not None
            receiver.stdin.write(sender_sdp + "\n")
            receiver.stdin.flush()

            receiver_sdp = _read_sdp(receiver, "receiver", deadline)

            assert sender.stdin is not None
            sender.stdin.write(receiver_sdp + "\n")
            sender.stdin.flush()

            # Drain remaining stdout on both sides so neither blocks on a
            # full pipe while the transfer finishes.
            sender_out: list[str] = []
            receiver_out: list[str] = []
            threading.Thread(
                target=_drain_stream, args=(sender.stdout, sender_out), daemon=True,
            ).start()
            threading.Thread(
                target=_drain_stream, args=(receiver.stdout, receiver_out), daemon=True,
            ).start()

            remaining = max(deadline - time.monotonic(), 1.0)
            try:
                sender.wait(timeout=remaining)
            except subprocess.TimeoutExpired as exc:
                raise E2EError("sender did not exit within timeout") from exc
            remaining = max(deadline - time.monotonic(), 1.0)
            try:
                receiver.wait(timeout=remaining)
            except subprocess.TimeoutExpired as exc:
                raise E2EError("receiver did not exit within timeout") from exc

            if sender.returncode != 0:
                raise E2EError(f"sender exited with code {sender.returncode}")
            if receiver.returncode != 0:
                raise E2EError(f"receiver exited with code {receiver.returncode}")

            if not out_path.is_file():
                raise E2EError(f"receiver did not create output file: {out_path}")

            want = _sha256(FIXTURE)
            got = _sha256(out_path)
            if want != got:
                raise E2EError(
                    f"checksum mismatch: fixture={want} received={got} "
                    f"(sizes: fixture={FIXTURE.stat().st_size} "
                    f"received={out_path.stat().st_size})"
                )

            print(
                f"ok  {FIXTURE.name}  ({FIXTURE.stat().st_size} B)  "
                f"sha256={want}"
            )
            return 0

        except E2EError as exc:
            print(f"e2e failed: {exc}", file=sys.stderr)
            if sender_err:
                print("--- sender stderr ---", file=sys.stderr)
                print("".join(sender_err), file=sys.stderr)
            if receiver_err:
                print("--- receiver stderr ---", file=sys.stderr)
                print("".join(receiver_err), file=sys.stderr)
            return 1
        finally:
            for p in (sender, receiver):
                if p is not None and p.poll() is None:
                    p.kill()
                    try:
                        p.wait(timeout=5)
                    except subprocess.TimeoutExpired:
                        pass


if __name__ == "__main__":
    sys.exit(run())
