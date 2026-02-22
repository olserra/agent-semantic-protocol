# Symplex Protocol

> **Semantic Interoperability Layer for AI Agents** — a lightweight extension of MCP (Model Context Protocol) built for the era of agentic AI meshes.

[![Go Report Card](https://goreportcard.com/badge/github.com/symplex-protocol/symplex)](https://goreportcard.com/report/github.com/symplex-protocol/symplex)
[![CI](https://github.com/symplex-protocol/symplex/actions/workflows/ci.yml/badge.svg)](https://github.com/symplex-protocol/symplex/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

> **Status:** Active development — v0.1 core is stable. See [docs/plans/](docs/plans/) for the current implementation roadmap.

---

## What Is Symplex?

Symplex enables AI agents to communicate **by meaning, not schema**.  Instead of exchanging rigid JSON tool calls, agents share **semantic intent vectors** — compact float32 embeddings that describe goals in a shared latent space.  Any agent that understands the vector space can negotiate, delegate, and collaborate — without pre-registered APIs.

### Core Concepts

| Concept | Description |
|---|---|
| **Intent Vector** | A `[]float32` embedding (e.g. 384-dim sentence-transformer) that encodes the *semantic goal* of a request |
| **Spontaneous Negotiation** | Agents advertise capabilities and bid on intents without a central broker |
| **Dynamic Discovery** | A TTL-based capability registry lets agents appear/disappear at runtime |
| **Federated Trust (DIDs)** | Each agent has a `did:symplex:<sha256(pubkey)>` identifier backed by an Ed25519 key-pair |
| **Distributed Workflows** | Accepted intents unfold into ordered workflow steps, dispatched across capable peers via libp2p |

---

## Comparison: Symplex vs. MCP vs. A2A

| Feature | MCP (Anthropic) | Google A2A | **Symplex** |
|---|---|---|---|
| Message format | JSON-RPC 2.0 | JSON (task objects) | **Protobuf wire + intent vectors** |
| Capability model | Static tool registry | Static skill cards | **Dynamic, TTL-based discovery** |
| Negotiation | None (direct call) | None | **Spontaneous cosine-similarity ranking** |
| Identity / trust | None | None | **Ed25519 DIDs, federated trust graph** |
| Transport | HTTP/SSE | HTTP | **libp2p (TCP/QUIC/WebRTC)** |
| Peer topology | Client–server | Client–server | **P2P mesh** |
| Semantic routing | No | No | **Yes (vector similarity)** |
| Distributed workflows | No | Task delegation only | **Yes (multi-step, multi-agent)** |

---

## Repository Layout

```
symplex/
├── proto/symplex.proto          # Protobuf message definitions (reference)
├── core/
│   ├── types.go                 # Go struct definitions
│   ├── encoding.go              # Protobuf wire encode/decode (no codegen required)
│   ├── did.go                   # DID generation, verification, trust graph
│   ├── handshake.go             # Cryptographic handshake protocol
│   ├── negotiation.go           # Intent negotiation + cosine ranking
│   ├── discovery.go             # Capability discovery registry
│   ├── encoding_test.go         # Unit tests (encoding, DID, cosine, discovery)
│   └── signing_test.go          # Unit tests (per-message Ed25519 signing)
├── p2p/
│   ├── host.go                  # libp2p AgentHost wrapper
│   ├── host_test.go             # Integration tests (handshake, intent, announce)
│   └── protocol.go              # WorkflowOrchestrator, convenience helpers
├── picoclaw/
│   └── client.go                # Adapter for Picoclaw AI assistant API
├── examples/
│   ├── simple-handshake/main.go # Two agents handshake over TCP
│   └── negotiation-demo/main.go # Full intent → negotiation → workflow loop
├── docs/
│   ├── spec.md                  # Protocol specification
│   └── plans/                   # Versioned implementation plans
└── .github/workflows/ci.yml    # GitHub Actions CI
```

---

## Quick Start

### Prerequisites

- Go 1.22+
- `git`

### Install & Run

```bash
git clone https://github.com/symplex-protocol/symplex
cd symplex

go mod download   # fetch all dependencies (including libp2p)

# Run the handshake demo
go run ./examples/simple-handshake/main.go

# Run the negotiation + workflow demo
go run ./examples/negotiation-demo/main.go
```

### Expected output (handshake demo)

```
╔══════════════════════════════════════════════╗
║     Symplex v0.1 — Simple Handshake Demo     ║
╚══════════════════════════════════════════════╝

✓ Agent Alpha started
  Peer ID : 12D3KooW...
  DID     : did:symplex:3a7f...
  Caps    : [nlp reasoning intent-parsing]

✓ Agent Beta started
  ...

── Performing Symplex Handshake ─────────────────────────────────────
[Beta] ← Handshake from "agent-alpha"  caps=[nlp reasoning intent-parsing]

[Alpha] ✓ Handshake complete!
  Peer caps  : [code-generation math vector-search storage]
  Protocol   : 1.0.0
```

### Run Tests

```bash
make test
# or: go test -v -race ./...
```

---

## Using Symplex in Your Go Project

```go
import (
    "github.com/symplex-protocol/symplex/core"
    "github.com/symplex-protocol/symplex/p2p"
)

// 1. Create an agent identity
agent, _ := core.NewAgent("my-agent", []string{"nlp", "summarisation"})

// 2. Start a P2P host
host, _ := p2p.NewHost(ctx, agent)

// 3. Register intent handler
host.OnIntent(func(peerID peer.ID, intent *core.IntentMessage) *core.NegotiationResponse {
    h := core.DefaultNegotiationHandler(agent)
    resp, _ := h(intent)
    return resp
})

// 4. Send an intent to a known peer
intent, _ := core.CreateIntent(agent,
    []float32{0.8, 0.2, 0.9}, // embedding
    []string{"summarisation"},
    "Summarise this document",
)
resp, _ := host.SendIntent(ctx, targetPeerID, intent)
```

### In-Process (No Network)

```go
bus := core.NewNegotiationBus()
bus.Register("agent-b", core.DefaultNegotiationHandler(agentB))

intent, _ := core.CreateIntent(agentA, vector, []string{"code-generation"}, payload)
resp, _ := bus.Negotiate("agent-b", intent)
```

### Picoclaw Integration

```go
client := picoclaw.NewClient("https://api.picoclaw.io",
    picoclaw.WithAPIKey("pk_..."),
    picoclaw.WithAgentID("my-picoclaw-agent"),
)
// Register as a Symplex negotiation handler
bus.Register("picoclaw", client.AsNegotiationHandler())
```

---

## Regenerate Protobuf Bindings (Optional)

The `core/` package uses `protowire` directly and does **not** require generated code.  If you want the generated `*.pb.go` files for gRPC or reflection:

```bash
make proto   # requires protoc + protoc-gen-go
```

---

## Contributing

1. Fork the repo and create a feature branch.
2. Run `make test` and `make lint` — all must pass.
3. Open a Pull Request with a clear description of the change.
4. Follow conventional commits (`feat:`, `fix:`, `docs:`, …).

Please read [docs/spec.md](docs/spec.md) before proposing wire-format changes — backward compatibility is critical.

---

## Roadmap

### Completed

| Version | Feature | Status |
|---------|---------|--------|
| v0.1 | Core protocol, handshake, negotiation, discovery, p2p transport | ✅ |
| v0.1 | Capability announcement handling in P2P (`MsgCapability` → DiscoveryRegistry) | ✅ |
| v0.1 | P2P integration tests (handshake, intent accept/reject, announce) | ✅ |
| v0.1 | Per-message Ed25519 signing (`IntentMessage`, `NegotiationResponse`) | ✅ |

### Planned

| Version | Feature |
|---------|---------|
| **v0.2** | QUIC transport (pending quic-go TLS session-ticket fix), WebRTC support |
| **v0.2** | Signature verification on receive (currently signed but not verified on inbound) |
| **v0.3** | Federated DID resolution (DID document over DHT), zk-SNARK capability proofs |
| **v1.0** | Stable wire format, MCP gateway adapter, multi-language SDKs (Python, TypeScript) |

See [docs/plans/](docs/plans/) for detailed implementation plans with exact file paths and test commands.

---

## License

MIT — see [LICENSE](LICENSE).
