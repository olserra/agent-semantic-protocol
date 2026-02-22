# Agent Semantic Protocol Protocol Specification — v0.1

**Status:** Draft
**Authors:** Agent Semantic Protocol Protocol Contributors
**Date:** 2025

---

## Table of Contents

1. [Overview](#1-overview)
2. [Goals and Non-Goals](#2-goals-and-non-goals)
3. [Architecture](#3-architecture)
4. [Message Format](#4-message-format)
5. [Protocol Flows](#5-protocol-flows)
6. [DID Trust Model](#6-did-trust-model)
7. [Capability Discovery](#7-capability-discovery)
8. [Semantic Routing](#8-semantic-routing)
9. [Distributed Workflows](#9-distributed-workflows)
10. [Transport Layer](#10-transport-layer)
11. [Picoclaw Integration](#11-picoclaw-integration)
12. [Security Considerations](#12-security-considerations)
13. [Future Extensions](#13-future-extensions)

---

## 1. Overview

Agent Semantic Protocol is a **semantic communication protocol** for AI agent meshes.  It extends the Model Context Protocol (MCP) with:

- **Vector-encoded intents** instead of rigid tool schemas
- **Spontaneous peer negotiation** without a central registry
- **Federated identity** via lightweight DID:agent-semantic-protocol identifiers
- **Distributed workflow orchestration** across capable peers

Agent Semantic Protocol does not replace MCP — it composes with it.  A Agent Semantic Protocol node can expose an MCP tool interface externally while using Agent Semantic Protocol internally for cross-agent coordination.

---

## 2. Goals and Non-Goals

### Goals

- Enable agents to express *semantic goals*, not just API calls
- Allow any agent to join or leave the mesh without reconfiguration
- Provide cryptographic identity with minimal setup (no blockchain required)
- Keep the wire format compact and language-neutral (Protobuf)

### Non-Goals (v0.1)

- Persistent message queuing
- Cross-mesh federation with other DID methods
- zk-proof capability verification
- Payments or resource metering

---

## 3. Architecture

```
  ┌───────────────────────────────────────────────────────┐
  │                   Agent Semantic Protocol Mesh                        │
  │                                                       │
  │   ┌──────────┐    intent    ┌──────────┐              │
  │   │ Agent A  │ ──────────► │ Agent B  │              │
  │   │ did:s:aa │ ◄────────── │ did:s:bb │              │
  │   └──────────┘  negotiate  └──────────┘              │
  │         │                       │                     │
  │         └─────── libp2p ────────┘                     │
  │                                                       │
  │   ┌──────────────────────────────────────────────┐   │
  │   │            Discovery Registry                │   │
  │   │  agentID → [capabilities]  (TTL-cached)      │   │
  │   └──────────────────────────────────────────────┘   │
  │                                                       │
  │   ┌──────────────────────────────────────────────┐   │
  │   │              Trust Graph                     │   │
  │   │  did:s:aa → did:s:bb : 0.72                  │   │
  │   └──────────────────────────────────────────────┘   │
  └───────────────────────────────────────────────────────┘
```

---

## 4. Message Format

All Agent Semantic Protocol messages are encoded in **Protobuf 3 binary format** (wire format compatible with `proto/agent-semantic-protocol.proto`).

Each message is **framed** before transmission:

```
┌─────────────────┬───────────┬──────────────────────────┐
│  Length (4B BE) │ Type (1B) │  Protobuf payload (N B)  │
└─────────────────┴───────────┴──────────────────────────┘
```

- **Length**: big-endian `uint32` = `1 + len(payload)` (includes type byte)
- **Type**: one of the `MessageType` constants below

### Message Types

| Hex  | Name                   | Direction            |
|------|------------------------|----------------------|
| 0x01 | `MsgHandshake`         | Bidirectional        |
| 0x02 | `MsgIntent`            | Requester → Provider |
| 0x03 | `MsgNegotiation`       | Provider → Requester |
| 0x04 | `MsgWorkflow`          | Orchestrator → Worker|
| 0x05 | `MsgCapability`        | Broadcast            |

### IntentMessage (type 0x02)

```protobuf
message IntentMessage {
  string          id            = 1;  // unique UUID / hex-random
  repeated float  intent_vector = 2;  // packed float32 semantic embedding
  repeated string capabilities  = 3;  // required capabilities
  string          did           = 4;  // sender DID
  string          payload       = 5;  // optional plain-text or JSON
  int64           timestamp     = 6;  // Unix ns
  float           trust_score   = 7;  // sender trust [0,1]
  map<string,string> metadata   = 8;
}
```

The **intent_vector** is the central primitive.  Agents embed natural-language goals using any sentence-encoder model (e.g. `all-MiniLM-L6-v2`, 384 dimensions).  The vector enables semantic matching without a shared ontology.

### HandshakeMessage (type 0x01)

```protobuf
message HandshakeMessage {
  string agent_id          = 1;
  string did               = 2;
  repeated string caps     = 3;
  string version           = 4;  // semver "1.0.0"
  int64  timestamp         = 5;
  bytes  public_key        = 6;  // Ed25519, 32 bytes
  bytes  challenge         = 7;  // 32-byte random nonce
  bytes  challenge_response= 8;  // Ed25519 sig of peer's challenge
}
```

### NegotiationResponse (type 0x03)

```protobuf
message NegotiationResponse {
  string          request_id      = 1;
  string          agent_id        = 2;
  bool            accepted        = 3;
  repeated string workflow_steps  = 4;
  string          did             = 5;
  repeated float  response_vector = 6;  // packed float32
  int64           timestamp       = 7;
  string          reason          = 8;
  float           trust_delta     = 9;  // suggested Δ to requester's trust
}
```

---

## 5. Protocol Flows

### 5.1 Handshake Flow

```
Initiator (A)                            Responder (B)
    │                                         │
    │── HandshakeMessage(challenge_A) ───────►│
    │   AgentID, DID, Capabilities, PubKey    │
    │   Challenge = rand(32)                  │
    │                                         │── ValidateDID(pubkey_A, did_A)
    │                                         │── Sign(challenge_A, privKey_B)
    │                                         │
    │◄── HandshakeMessage(challenge_B) ───────│
    │   AgentID, DID, Capabilities, PubKey    │
    │   Challenge = rand(32)                  │
    │   ChallengeResponse = Sig(challenge_A)  │
    │                                         │
    │── ValidateDID(pubkey_B, did_B) ─────────│
    │── VerifySig(challenge_A, Sig, pubKey_B) │
    │                                         │
    │ [Both agents register each other in     │
    │  DiscoveryRegistry]                     │
```

### 5.2 Intent Negotiation Flow

```
Agent A                                   Agent B
    │                                         │
    │── IntentMessage ────────────────────────►│
    │   intent_vector=[0.8, 0.2, 0.9, ...]    │
    │   capabilities=["code-gen","python"]    │
    │   payload="generate sort function"      │
    │                                         │── EvaluateCapabilities()
    │                                         │── CosineSimilarity(v, myVec)
    │                                         │── BuildWorkflow()
    │                                         │
    │◄── NegotiationResponse ─────────────────│
    │   accepted=true                         │
    │   workflow_steps=[step1, step2, ...]    │
    │   trust_delta=+0.05                     │
    │                                         │
    │── UpdateTrustGraph(B.DID, +0.05) ───────│
```

### 5.3 Distributed Workflow Execution

```
Orchestrator                  Worker_1          Worker_2
     │                            │                 │
     │── WorkflowMsg(step-1) ────►│                 │
     │                            │── execute()     │
     │                            │◄─ result        │
     │                            │                 │
     │── WorkflowMsg(step-2) ─────────────────────►│
     │                                              │── execute()
     │                                              │◄─ result
     │◄──────────── aggregated results ─────────────│
```

---

## 6. DID Trust Model

### 6.1 DID Format

```
did:agent-semantic-protocol:<hex(sha256(ed25519_public_key))>
```

Example: `did:agent-semantic-protocol:3a7fc29e8f1b4d5a9e0c3b6d7f2a4e8c1d5b9f3a7e2c0d4b6a8f1e3d5c7b9f0`

### 6.2 Key Generation

Each agent generates a fresh Ed25519 key-pair on first start.  The DID is derived deterministically from the public key — no registration required.

```
privKey, pubKey  = ed25519.GenerateKey(rand.Reader)
did              = "did:agent-semantic-protocol:" + hex(sha256(pubKey))
```

### 6.3 DID Binding Verification

Upon receiving a HandshakeMessage:

1. Parse the `did` field → extract the hex ID.
2. Compute `sha256(incoming.PublicKey)`.
3. Assert `hex(hash) == did.ID`.

If the assertion fails, the connection is rejected.

### 6.4 Trust Graph

Trust is stored as directed edge weights `T(from_DID, to_DID) ∈ [0.0, 1.0]`.

- Initial trust = `0.5` (neutral)
- Every NegotiationResponse carries a `trust_delta`
- Accepted intents: `Δ = +0.05`; rejected: `Δ = −0.02`
- Values are clamped to `[0.0, 1.0]`

Future versions will propagate trust transitively across the mesh.

---

## 7. Capability Discovery

Agents announce capabilities via `CapabilityAnnouncement` messages broadcast to connected peers.  Announcements have a TTL (seconds); `TTL=0` means permanent.

The local `DiscoveryRegistry` indexes profiles by `AgentID` and supports:
- `FindByCapability(required ...string) []AgentProfile`
- `FindByDID(did string) (AgentProfile, bool)`
- Automatic TTL eviction via background goroutine

### Discovery on Handshake

Capability exchange is **embedded in the handshake** — no separate announcement needed for agents that are directly connected.  Broadcasts serve agents in multi-hop topologies.

---

## 8. Semantic Routing

When multiple peers satisfy the required capabilities, Agent Semantic Protocol ranks them by **cosine similarity** between the intent vector and each peer's registered embedding vector:

```
similarity(a, b) = (a · b) / (‖a‖ · ‖b‖)
```

The peer with the highest similarity is tried first.

```
RankCandidates(intentVector []float32, candidates []AgentProfile) []AgentProfile
```

If no embedding is registered for a peer, it is ranked last (score = 0).

---

## 9. Distributed Workflows

A `NegotiationResponse` with `accepted=true` includes a `workflow_steps` slice.  Each step is an opaque string by convention:

```
"step-1: parse_intent"
"step-2: generate_code[python]"
"step-3: validate_output"
"step-4: stream_result"
```

The `WorkflowOrchestrator` in `p2p/protocol.go` dispatches steps to peers concurrently using goroutines, collecting results into `[]StepResult`.

### Concurrency Model

```
goroutine per step
     │── SendIntent(bestPeer, stepIntent)
     │── collect NegotiationResponse
     │── write to results[i] under sync.Mutex
```

Steps within a workflow are executed **concurrently by default**.  Sequential dependencies can be encoded in step payloads or handled by a stateful orchestrator built on top.

---

## 10. Transport Layer

Agent Semantic Protocol uses **libp2p** (`/agent-semantic-protocol/1.0.0`) as the transport protocol.

- **Default**: TCP (`/ip4/x.x.x.x/tcp/N`)
- **Planned**: QUIC, WebRTC for browser agents
- **Stream multiplexing**: yamux (libp2p default)
- **Encryption**: Noise protocol (libp2p default)

For testing and in-process simulation, `core.NegotiationBus` provides a zero-network channel-based implementation.

---

## 11. Picoclaw Integration

The `picoclaw` package provides an HTTP/JSON adapter that bridges Agent Semantic Protocol with Picoclaw-compatible AI assistant services.

```
Agent Semantic Protocol IntentMessage
        │
        ▼
picoclaw.Client.SendIntent()
        │
        ▼
POST /v1/intent  ─────► Picoclaw API
                ◄─────  IntentResponse
        │
        ▼
core.NegotiationResponse
```

The client implements `core.NegotiationHandler` and can be directly registered on a `NegotiationBus`:

```go
bus.Register("picoclaw", client.AsNegotiationHandler())
```

---

## 12. Security Considerations

| Threat | Mitigation (v0.1) |
|--------|-------------------|
| DID spoofing | DID/key binding verified on every handshake |
| Man-in-the-middle | Noise protocol encryption via libp2p |
| Intent flooding | Trust graph penalises rejected intents |
| Sybil attacks | Ed25519 key generation is cheap; federation and staking planned for v0.3 |
| Replay attacks | Timestamp field; monotonic nonce planned for v0.2 |

**Per-message signatures** (Ed25519 over the entire Protobuf payload) are the primary planned improvement for v0.2.

---

## 13. Future Extensions

| Version | Feature |
|---------|---------|
| v0.2 | Per-message Ed25519 signatures, QUIC transport, WASM runtime |
| v0.3 | DID resolution over libp2p DHT, zk-SNARK capability proofs |
| v0.4 | Cross-mesh federation (did:web, did:key interop) |
| v1.0 | Stable wire format, MCP gateway adapter, streaming intents |
| v1.x | Multi-language SDKs (Python, TypeScript, Rust) |
