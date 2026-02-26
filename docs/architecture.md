# Agent Semantic Protocol Architecture

## System Overview

The Agent Semantic Protocol (ASP) is designed to enable seamless, secure, and efficient communication between AI agents. The architecture is composed of five key layers:

1. **Identity**: Decentralized Identifiers (DIDs) ensure agent authenticity and trust.
2. **Transport**: Multi-transport support (TCP, QUIC, WebRTC) with Noise protocol encryption.
3. **Discovery**: Capability registry for dynamic agent discovery.
4. **Negotiation**: Semantic intent exchange and capability matching.
5. **Workflow**: Distributed workflows across multiple agents.

---

## Layer Diagram

```
+-------------------+
|   Application     |
+-------------------+
|     Workflow      |
+-------------------+
|    Negotiation    |
+-------------------+
|     Discovery     |
+-------------------+
|     Transport     |
+-------------------+
|      Identity     |
+-------------------+
```

---

## Data Flow: Intent Lifecycle

1. **Intent Creation**: An agent creates an `IntentMessage` with a semantic vector and required capabilities.
2. **Capability Matching**: The receiving agent evaluates the intent against its capabilities using cosine similarity.
3. **Negotiation**: If compatible, the agents negotiate the terms of the workflow.
4. **Workflow Execution**: The intent is decomposed into steps and executed across agents.

---

## Key Data Structures

- **IntentMessage**: Encodes the semantic goal of a request.
- **NegotiationResponse**: Communicates acceptance or rejection of an intent.
- **Agent**: Represents an agent’s identity and capabilities.

---

## Module Dependency Graph

```
+-------------------+
|       Core        |
|-------------------|
| types.go          |
| encoding.go       |
| negotiation.go    |
+-------------------+
        ^
        |
+-------------------+
|       P2P         |
|-------------------|
| host.go           |
| protocol.go       |
+-------------------+
```

---

## Extension Points

- **Transports**: Add new transport protocols by extending the `Transport` interface.
- **Capability Matchers**: Implement custom matching logic for specific use cases.

---

This document provides a high-level overview of the architecture. For more details, refer to the source code and additional documentation in the `docs/` directory.
