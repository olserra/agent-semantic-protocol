# Agent Semantic Protocol Roadmap Execution Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Close the most impactful gaps in Agent Semantic Protocol v0.1: fix the capability announcement TODO, add p2p integration tests, enable per-message Ed25519 signing, and update documentation.

**Architecture:** Changes are additive and backward-compatible. Tasks 1–3 are low-risk correctness fixes; Task 4 (signing) touches the wire format but only for new messages. No API breakage.

**Tech Stack:** Go 1.22+, libp2p v0.36.0, Ed25519 (stdlib `crypto/ed25519`), standard `testing` package.

---

## Priority Order

| # | Task | Effort | Impact |
|---|------|--------|--------|
| 1 | Fix capability announcement TODO | XS | Correctness |
| 2 | p2p integration tests | M | Coverage 52% → 75%+ |
| 3 | Per-message Ed25519 signing | M | Security |
| 4 | QUIC transport | S | Transport coverage |
| 5 | README update | XS | Discoverability |
| 6 | Benchmarks | M | Performance visibility |
| 7 | Python/TypeScript SDKs | XL | Ecosystem |

---

## Task 1: Fix capability announcement TODO

**Files:**
- Modify: `p2p/host.go:292-296`

**Context:** `handleIncomingCapability` receives a `CapabilityAnnouncement` over the wire but ignores it. The decoder `core.DecodeCapabilityAnnouncement` and registrar `registry.AnnounceFromMessage` already exist.

**Step 1: Verify the decoder exists**

Run: `grep -n "DecodeCapabilityAnnouncement\|AnnounceFromMessage" /Users/olserra-duvenbeck/Developer/agent-semantic-protocol/core/*.go`
Expected: both symbols are defined.

**Step 2: Replace the TODO body**

Replace in `p2p/host.go`:
```go
func (ah *AgentHost) handleIncomingCapability(data []byte) {
	ann, err := core.DecodeCapabilityAnnouncement(data)
	if err != nil {
		return
	}
	ah.discovery.AnnounceFromMessage(ann)
}
```

**Step 3: Build**

Run: `go build ./...`
Expected: clean.

**Step 4: Commit**

```bash
git add p2p/host.go
git commit -m "fix: register incoming CapabilityAnnouncement in discovery registry"
```

---

## Task 2: p2p integration tests

**Files:**
- Create: `p2p/host_test.go`

**Context:** Zero test coverage for `p2p/`. We need real libp2p connections, so tests spin up two in-process `AgentHost` instances that connect over loopback.

**Step 1: Write the failing tests**

File: `p2p/host_test.go`
```go
package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/olserra/agent-semantic-protocol/core"
	"github.com/olserra/agent-semantic-protocol/p2p"
)

func makeAgent(t *testing.T, id string, caps []string) *core.Agent {
	t.Helper()
	a, err := core.NewAgent(id, caps)
	if err != nil {
		t.Fatalf("NewAgent(%q): %v", id, err)
	}
	return a
}

func makeHost(t *testing.T, agent *core.Agent) *p2p.AgentHost {
	t.Helper()
	ctx := context.Background()
	h, err := p2p.NewHost(ctx, agent)
	if err != nil {
		t.Fatalf("NewHost: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

// TestHandshake verifies that two agents can complete the Agent Semantic Protocol handshake.
func TestHandshake(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp"})
	beta  := makeAgent(t, "beta",  []string{"code-gen"})

	hA := makeHost(t, alpha)
	hB := makeHost(t, beta)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hA.Connect(ctx, hB.AddrInfo()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	resp, err := hA.Handshake(ctx, hB.PeerID())
	if err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	if resp.AgentID != "beta" {
		t.Errorf("AgentID: got %q want %q", resp.AgentID, "beta")
	}
	if len(resp.Capabilities) == 0 {
		t.Error("expected non-empty capabilities in handshake response")
	}
}

// TestSendIntent verifies intent send/receive and negotiation response over libp2p.
func TestSendIntent(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp"})
	beta  := makeAgent(t, "beta",  []string{"summarisation"})

	hA := makeHost(t, alpha)
	hB := makeHost(t, beta)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hA.Connect(ctx, hB.AddrInfo()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	intent, err := core.CreateIntent(alpha,
		[]float32{0.9, 0.1, 0.5},
		[]string{"summarisation"},
		"summarise this doc",
	)
	if err != nil {
		t.Fatalf("CreateIntent: %v", err)
	}

	resp, err := hA.SendIntent(ctx, hB.PeerID(), intent)
	if err != nil {
		t.Fatalf("SendIntent: %v", err)
	}

	if !resp.Accepted {
		t.Errorf("expected intent accepted, got reason: %s", resp.Reason)
	}
}

// TestIntentRejected verifies rejection when a peer lacks required capabilities.
func TestIntentRejected(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp"})
	beta  := makeAgent(t, "beta",  []string{"code-gen"}) // does NOT have summarisation

	hA := makeHost(t, alpha)
	hB := makeHost(t, beta)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hA.Connect(ctx, hB.AddrInfo()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	intent, _ := core.CreateIntent(alpha, []float32{0.5, 0.5}, []string{"summarisation"}, "")

	resp, err := hA.SendIntent(ctx, hB.PeerID(), intent)
	if err != nil {
		t.Fatalf("SendIntent: %v", err)
	}

	if resp.Accepted {
		t.Error("expected intent to be rejected, was accepted")
	}
}

// TestAnnounceCapabilities verifies that AnnounceCapabilities registers the
// announcing agent in the receiver's discovery registry.
func TestAnnounceCapabilities(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp", "reasoning"})
	beta  := makeAgent(t, "beta",  []string{"code-gen"})

	hA := makeHost(t, alpha)
	hB := makeHost(t, beta)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hB.Connect(ctx, hA.AddrInfo()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	hA.AnnounceCapabilities(ctx)
	time.Sleep(200 * time.Millisecond) // allow async streams to complete

	found := hB.Discovery().FindByCapability("nlp")
	if len(found) == 0 {
		t.Error("expected alpha to be discoverable by beta after AnnounceCapabilities")
	}
}
```

**Step 2: Run tests to confirm they fail (or compile-fail on missing types)**

Run: `go test -v -race -timeout 30s ./p2p/...`
Expected: either FAIL or compilation error — not PASS.

**Step 3: Fix any compilation issues, then run again**

Run: `go test -v -race -timeout 30s ./p2p/...`
Expected: all 4 tests PASS.

**Step 4: Confirm full suite still passes**

Run: `go test -race ./...`
Expected: PASS.

**Step 5: Commit**

```bash
git add p2p/host_test.go
git commit -m "test: add p2p integration tests for handshake, intent send/receive, and capability announcement"
```

---

## Task 3: Per-message Ed25519 signing

**Files:**
- Modify: `core/types.go` — add `Signature []byte` field to `IntentMessage` and `NegotiationResponse`
- Modify: `core/encoding.go` — encode/decode the new field (field numbers 10 / 9)
- Modify: `core/negotiation.go` — sign outgoing messages in `CreateIntent` and `DefaultNegotiationHandler`
- Modify: `core/did.go` — expose `SignMessage` / `VerifyMessage` helpers (if not present)
- Create: `core/signing_test.go` — tests for sign + verify round-trip

**Context:** `core.DID` already has `Sign(msg []byte)` and `Verify(msg []byte, sig []byte)` methods. We extend the wire format (new protobuf field, backward-compatible — zero value = unsigned) and sign the serialised payload before writing the Signature field.

**Step 1: Write the failing test**

File: `core/signing_test.go`:
```go
package core_test

import (
	"testing"
	"github.com/olserra/agent-semantic-protocol/core"
)

func TestIntentMessageSigning(t *testing.T) {
	agent, err := core.NewAgent("signer", []string{"nlp"})
	if err != nil {
		t.Fatal(err)
	}

	intent, err := core.CreateIntent(agent, []float32{0.5, 0.5}, []string{"nlp"}, "hello")
	if err != nil {
		t.Fatal(err)
	}

	if len(intent.Signature) == 0 {
		t.Error("CreateIntent should set a non-empty Signature")
	}

	if !agent.DID.Verify([]byte(intent.ID+intent.Payload), intent.Signature) {
		t.Error("Signature failed to verify against agent DID")
	}
}

func TestNegotiationResponseSigning(t *testing.T) {
	agent, err := core.NewAgent("responder", []string{"code-gen"})
	if err != nil {
		t.Fatal(err)
	}

	intent := &core.IntentMessage{
		ID:           "test-id",
		Capabilities: []string{"code-gen"},
		DID:          agent.DID.String(),
	}

	h := core.DefaultNegotiationHandler(agent)
	resp, err := h(intent)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Signature) == 0 {
		t.Error("NegotiationResponse should carry a Signature")
	}
}
```

**Step 2: Run test — confirm it fails**

Run: `go test -v -run TestIntentMessageSigning ./core/...`
Expected: FAIL — `Signature` field does not exist.

**Step 3: Add Signature field to types**

In `core/types.go`, add to `IntentMessage`:
```go
Signature []byte
```
And to `NegotiationResponse`:
```go
Signature []byte
```

**Step 4: Add encoding for Signature**

In `core/encoding.go`, add field 10 for `IntentMessage.Signature` (bytes, tag `(10 << 3) | 2`) and field 9 for `NegotiationResponse.Signature`.

**Step 5: Sign in CreateIntent**

In `core/negotiation.go`, after building the `IntentMessage`:
```go
sigPayload := []byte(intent.ID + intent.Payload)
sig, err := sender.DID.Sign(sigPayload)
if err != nil {
    return nil, fmt.Errorf("CreateIntent: sign: %w", err)
}
intent.Signature = sig
```

**Step 6: Sign in DefaultNegotiationHandler**

After building the `NegotiationResponse`, sign `resp.RequestID + resp.Reason`:
```go
sig, err := agent.DID.Sign([]byte(resp.RequestID + resp.Reason))
if err == nil {
    resp.Signature = sig
}
```

**Step 7: Run tests**

Run: `go test -race ./...`
Expected: all tests PASS, including the new signing tests.

**Step 8: Commit**

```bash
git add core/types.go core/encoding.go core/negotiation.go core/signing_test.go
git commit -m "feat: add per-message Ed25519 signing to IntentMessage and NegotiationResponse"
```

---

## Task 4: QUIC transport (optional, low-risk)

**Files:**
- Modify: `p2p/host.go:54-57` — add QUIC listen address alongside TCP

**Context:** libp2p already has QUIC support via `go-libp2p-quic-transport` which is included transitively. Adding a second `ListenAddrStrings` entry enables it.

**Step 1: Update NewHost**

Replace:
```go
libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
```
With:
```go
libp2p.ListenAddrStrings(
    "/ip4/127.0.0.1/tcp/0",
    "/ip4/127.0.0.1/udp/0/quic-v1",
),
```

**Step 2: Build and test**

Run: `go test -race ./...`
Expected: all tests PASS (p2p tests should still work; libp2p negotiates transport automatically).

**Step 3: Commit**

```bash
git add p2p/host.go
git commit -m "feat: enable QUIC transport alongside TCP in AgentHost"
```

---

## Task 5: README update

**Files:**
- Modify: `README.md`

**Changes:**
- Update Roadmap section: mark v0.2 items in progress / completed as tasks land
- Add "Test Coverage" badge or note
- Add a "Development Status" section showing what's complete vs. planned
- Mention the `docs/plans/` directory for contributors

---

## Task 6: Benchmarks (future)

**Files:**
- Create: `core/bench_test.go`
- Create: `p2p/bench_test.go`

**Benchmarks to add:**
- `BenchmarkCosineSimilarity` — 384-dim vectors, N iterations
- `BenchmarkIntentEncodeDecode` — round-trip time
- `BenchmarkNegotiationBus` — in-process negotiate throughput
- `BenchmarkP2PHandshake` — full libp2p handshake latency

Run with: `go test -bench=. -benchmem ./...`

---

## Task 7: Python SDK (future)

**Files:** New repo `agent-semantic-protocol-python/` or `sdk/python/`

**Minimum viable SDK:**
- `agent-semantic-protocol.Agent` — wraps DID generation (use `cryptography` lib for Ed25519)
- `agent-semantic-protocol.IntentMessage` — manual protobuf encoding matching Go wire format
- `agent-semantic-protocol.connect(addr)` — TCP socket, framing protocol
- `agent-semantic-protocol.negotiate(intent)` — send + receive

**Reference implementation:** `core/encoding.go` (protobuf field numbers are the source of truth).
