# Agent Semantic Protocol

> **Connecting AI agents through meaning, not schema.**

![Go Report Card](https://goreportcard.com/badge/github.com/olserra/agent-semantic-protocol)
[![CI](https://github.com/olserra/agent-semantic-protocol/actions/workflows/ci.yml/badge.svg)](https://github.com/olserra/agent-semantic-protocol/actions/workflows/ci.yml)
[![Codecov](https://codecov.io/gh/olserra/agent-semantic-protocol/branch/main/graph/badge.svg)](https://codecov.io/gh/olserra/agent-semantic-protocol)
[![Go Version](https://img.shields.io/github/go-mod/go-version/olserra/agent-semantic-protocol)](https://golang.org/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## Why Agent Semantic Protocol?

MCP connects LLMs to tools. **ASP connects agents to agents.**

ASP enables AI agents to communicate **by meaning, not schema**. Instead of rigid JSON APIs, agents share **semantic intent vectors** — compact embeddings that describe goals in a shared latent space. This allows:

- **Dynamic Discovery**: Agents appear/disappear at runtime.
- **Spontaneous Negotiation**: No central broker needed.
- **Federated Trust**: Decentralized identifiers (DIDs) ensure authenticity.

---

## Core Concepts

| Concept | Description |
|---|---|
| **Intent Vector** | A `[]float32` embedding that encodes the *semantic goal* of a request |
| **Spontaneous Negotiation** | Agents advertise capabilities and bid on intents |
| **Dynamic Discovery** | TTL-based capability registry |
| **Federated Trust (DIDs)** | Self-sovereign identity with Ed25519 keys |
| **Distributed Workflows** | Multi-step workflows across peers |

---

## Architecture

```
+-------------------+   +-------------------+
|   Agent Alpha     |   |   Agent Beta      |
|-------------------|   |-------------------|
| Capabilities: NLP|   | Capabilities: Math|
|-------------------|   |-------------------|
| Intent Vector --> |   | <-- Negotiation   |
+-------------------+   +-------------------+
```

---

## Quickstart

### Prerequisites

- Go 1.22+
- `git`

### Install & Run

```bash
git clone https://github.com/olserra/agent-semantic-protocol.git
cd agent-semantic-protocol
go mod download

# Run the handshake demo
go run ./examples/simple-handshake/main.go
```

---

## Roadmap

See [ROADMAP.md](ROADMAP.md) for details.

---

## Contributing

We welcome contributions! See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

---

## License

MIT — see [LICENSE](LICENSE).
