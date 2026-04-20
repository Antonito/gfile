# gfile Protocol

gfile transfers a file between two peers over one or more WebRTC data channels.
A STUN server helps the two peers find each other through NAT; the file itself
never transits a third party. STUN can be disabled (`--stun=""`) to fall back
to host/mDNS candidates only — useful on a LAN where no public reflexive
candidate is needed. Host candidates are advertised as `.local` mDNS hostnames
by default (toggle with `--mdns=false`). This document describes the wire
format and the two transfer modes (single-PC and multi-PC).

## Signaling

gfile does not ship a signaling server. Peers exchange SDP out-of-band:

1. The sender prints a base64-encoded offer SDP to stdout.
2. The user copies the offer to the receiver, who prints a base64-encoded
   answer SDP.
3. The user copies the answer back to the sender.

Once the peer connection is up, all further communication happens on the
data channels described below.

## Session lifecycle

```
sender                                            receiver
  │                                                   │
  │─── offer SDP ─────────────────────────────────▶   │
  │◀── answer SDP ────────────────────────────────    │
  │                                                   │
  │═══ data channel open (label=primary) ═════════    │
  │                                                   │
  │─── METADATA (version, codec, size, sha256) ──▶    │
  │─── DATA  (offset=0,       payload) ──────────▶    │
  │─── DATA  (offset=256K,    payload) ──────────▶    │
  │            ...                                    │
  │─── DATA  (offset=N,       final)   ──────────▶    │
  │─── EOF ──────────────────────────────────────▶    │
  │                                                   │
  │                     (receiver verifies sha256)    │
  │                                                   │
```

Either side may send `ABORT` at any time to terminate the transfer.

## Wire format

Each data channel message carries exactly one frame — there is no
inter-frame delimiter. The first byte of every frame is the frame type.

Multi-byte integers are big-endian.

| Byte   | Name                 | Body layout                             | Direction  |
| ------ | -------------------- | --------------------------------------- | ---------- |
| `0x01` | `METADATA`           | `version(1) codec(1) size(8) sha256(32)` | S → R      |
| `0x02` | `DATA`               | `offset(8) payload(...)`                | S → R      |
| `0x03` | `EOF`                | (empty) — single-PC only                | S → R      |
| `0x04` | `ABORT`              | `reason(utf-8)`                         | either way |
| `0x05` | `ADD_PEER_OFFER`     | `peer_id(1) sdp_len(4) sdp(...)` — multi-PC | S → R  |
| `0x06` | `ADD_PEER_ANSWER`    | `peer_id(1) sdp_len(4) sdp(...)` — multi-PC | R → S  |
| `0x07` | `TRANSFER_COMPLETE`  | (empty) — multi-PC only                 | S → R      |

### `METADATA`

Total frame size: 43 bytes.

```
 0      1       2       3              11                             43
 ┌──────┬───────┬───────┬──────────────┬──────────────────────────────┐
 │ type │ ver   │ codec │  file_size   │           sha256             │
 │ 0x01 │  (1)  │  (1)  │   (8 BE)     │           (32)               │
 └──────┴───────┴───────┴──────────────┴──────────────────────────────┘
```

- `version`: current version is `0x01`. Receivers reject other values.
- `codec`: `0x00` = raw bytes; `0x01` = each DATA payload is an independent
  zstd frame.
- `file_size`: byte count of the **decompressed** file.
- `sha256`: SHA-256 of the **decompressed** file. The receiver verifies it
  after EOF / TRANSFER_COMPLETE.

### `DATA`

```
 0      1                      9                 9+N
 ┌──────┬──────────────────────┬──────────────────┐
 │ type │       offset         │    payload       │
 │ 0x02 │       (8 BE)         │       (N)        │
 └──────┴──────────────────────┴──────────────────┘
```

- `offset`: byte offset into the decompressed file where `payload` begins.
- `payload`: up to `ChunkSize = 256 KiB`. The final chunk may be smaller or
  empty. Under `CodecZstd` the payload is an independent zstd frame whose
  decompressed length covers `[offset, offset + decompressed_len)`.

### `EOF`, `TRANSFER_COMPLETE`

Single-byte frames — just the type byte, no body. `EOF` terminates a
single-PC transfer; `TRANSFER_COMPLETE` terminates a multi-PC transfer on the
control channel.

### `ABORT`

```
 0      1                         N
 ┌──────┬─────────────────────────┐
 │ type │       reason            │
 │ 0x04 │       (utf-8)           │
 └──────┴─────────────────────────┘
```

Receiver closes the output file, removes any partial bytes on disk, and
surfaces `abort: <reason>` to the user. Sender does the symmetric thing and
stops sending.

### `ADD_PEER_OFFER`, `ADD_PEER_ANSWER`

Multi-PC only. Carried on the control channel to negotiate additional data
peer connections.

```
 0      1       2               6                    6+sdp_len
 ┌──────┬───────┬───────────────┬───────────────────────┐
 │ type │peer_id│   sdp_len     │         sdp           │
 │ 0x05 │  (1)  │    (4 BE)     │                       │
 └──────┴───────┴───────────────┴───────────────────────┘
```

- `peer_id`: the data PC index (`0..connections-1`).
- `sdp`: base64 SDP as a UTF-8 string.

## Single-PC mode

One peer connection, one data channel labeled `primary`. The sender sends
`METADATA`, streams `DATA` frames in ascending offset, and ends with `EOF`.
The receiver writes each payload to the output file at its advertised
offset, then verifies the SHA-256.

## Multi-PC mode

A single WebRTC data channel has a throughput ceiling that often sits well
below the physical link — one connection, one in-flight window, one
send/receive pipeline. Multi-PC works around this by opening several peer
connections in parallel and striping the file across them. The user asks
for it with `--connections N` (1..16); `N=1` is the default and keeps the
single-PC wire format.

In this mode, one **control** peer connection carries orchestration, and
N **data** peer connections carry file bytes in parallel.

```
            ┌─── control  (label=control)  ─── METADATA, ADD_PEER_*,
            │                                  TRANSFER_COMPLETE, ABORT
 sender  ───┤
            ├─── data-0   (label=data-0)   ─── DATA …
            ├─── data-1   (label=data-1)   ─── DATA …
            │    …
            └─── data-N-1 (label=data-N-1) ─── DATA …
```

### Negotiation

1. Sender creates the control PC first and exchanges the primary offer/answer
   out-of-band.
2. Sender sends `METADATA` on the control channel.
3. For each data peer `i = 0..N-1`, sender creates a new PC, builds a local
   offer, and sends it as `ADD_PEER_OFFER(peer_id=i, sdp)` on the control
   channel.
4. Receiver creates a matching PC, returns `ADD_PEER_ANSWER(peer_id=i, sdp)`
   on the control channel.
5. Once all data channels are open, the sender starts striping `DATA` frames
   across them. Each DATA frame is self-contained (it carries its own
   offset), so ordering between channels does not matter.
6. When the sender has flushed every data channel, it sends
   `TRANSFER_COMPLETE` on the control channel. The receiver then verifies the
   SHA-256.

`ABORT` on any channel terminates the whole transfer.

## Versioning

The current `ProtocolVersion` is `0x01`. Receivers reject a `METADATA` frame
with any other version by sending `ABORT`. Future breaking changes bump this
byte; backward-compatible additions (e.g. a new codec value) do not.
