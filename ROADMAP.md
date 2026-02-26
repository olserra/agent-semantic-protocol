# Agent Semantic Protocol Roadmap

## Completed (v0.1)

- Core protocol: handshake, negotiation, discovery, p2p transport.
- Capability announcement handling in P2P (`MsgCapability` → DiscoveryRegistry).
- P2P integration tests (handshake, intent accept/reject, announce).
- Per-message Ed25519 signing (`IntentMessage`, `NegotiationResponse`).
- Observability tools for debugging and monitoring.
- Auditable logs for secure and compliant message tracking.

---

## In Progress (v0.2)

- QUIC transport support (pending quic-go TLS session-ticket fix).
- Signature verification on receive (currently signed but not verified on inbound).
- Benchmarks for key operations (e.g., handshake, intent processing).

---

## Planned (v0.3 – v1.0)

- Federated DID resolution (DID document over DHT).
- zk-SNARK capability proofs for privacy-preserving workflows.
- Stable wire format and MCP gateway adapter.
- Multi-language SDKs (Python, TypeScript).
- Comprehensive documentation updates.

---

## Call for Contributors

We need your help to make Agent Semantic Protocol the gold standard for AI agent communication! Here’s where you can contribute:

- **Multi-Language SDKs**: Help us build Python and TypeScript SDKs.
- **Advanced Features**: Contribute to federated DID resolution, zk-SNARK proofs, or distributed workflows.
- **Performance Benchmarks**: Add benchmarks to showcase the protocol’s efficiency.
- **Documentation**: Improve guides, tutorials, and examples.

Check out the [CONTRIBUTING.md](CONTRIBUTING.md) for details on how to get started!
