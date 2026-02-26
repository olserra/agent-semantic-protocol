# How Agent Semantic Protocol Works

## The Problem

Existing protocols for AI communication, such as JSON-RPC or HTTP APIs, rely on rigid schemas and predefined endpoints. This limits flexibility and interoperability in dynamic, decentralized environments where agents need to collaborate without prior coordination.

---

## Intent Vectors

### What Are They?

Intent vectors are `[]float32` embeddings that encode the semantic goal of a request. For example:

- **Input**: "Summarize this document."
- **Intent Vector**: `[0.8, 0.2, 0.9, ...]` (384 dimensions)

### Why Do They Work?

- **Compact Representation**: Encodes meaning in a shared latent space.
- **Language-Agnostic**: Works across different programming languages and frameworks.
- **Flexible Matching**: Enables cosine similarity-based capability matching.

---

## The Negotiation Flow

1. **Discovery**: Agents broadcast their capabilities using a TTL-based registry.
2. **Intent Creation**: An agent formulates an `IntentMessage` with a semantic vector and required capabilities.
3. **Capability Matching**: The receiving agent evaluates the intent against its capabilities using cosine similarity.
4. **Negotiation**: If compatible, the agents negotiate the terms of the workflow.
5. **Workflow Execution**: The intent is decomposed into steps and executed across agents.

---

## Trust and Identity

### Decentralized Identifiers (DIDs)

Each agent is identified by a DID, such as:

```
did:agent-semantic-protocol:3a7f...
```

- **Ed25519 Keys**: Ensure authenticity and integrity.
- **Self-Sovereign**: No central authority required.

### Trust Delta

Agents maintain a trust score for peers, updated based on interactions. This incentivizes honest behavior and penalizes malicious actors.

---

## Distributed Workflows

### How Steps Compose

1. **Decomposition**: The intent is broken into discrete workflow steps.
2. **Assignment**: Steps are assigned to agents based on their capabilities.
3. **Execution**: Each agent executes its assigned steps and reports results.

### Example

- **Intent**: "Translate and summarize this document."
- **Workflow**:
  1. Agent A: Translate the document.
  2. Agent B: Summarize the translated text.

---

## FAQ

### Why not use HTTP APIs?

HTTP APIs require predefined schemas and endpoints, which limit flexibility. ASP’s semantic approach enables dynamic, decentralized communication.

### How is security ensured?

- **Ed25519 Signing**: Ensures message authenticity.
- **Noise Protocol**: Encrypts transport-level communication.
- **DID Binding**: Authenticates agents.

### Can I use ASP with non-Go languages?

Yes! Multi-language SDKs (Python, TypeScript) are planned to make ASP accessible to a broader audience.

---

This document provides a conceptual overview of how ASP works. For technical details, see the [architecture](architecture.md) document.
