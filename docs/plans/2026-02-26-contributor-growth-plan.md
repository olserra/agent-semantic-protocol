# Contributor Growth Plan — Agent Semantic Protocol

**Date:** 2026-02-26
**Goal:** Maximize open-source contributor traction — on par with or better than MCP
**Strategy:** Reduce friction, increase clarity, show direction, reward participation

---

## 0. Rename Leftovers (Symplex → ASP)

Old name "Symplex" still appears in two source files that were missed in the previous rename.

| File | Line | Current | Fix |
|------|------|---------|-----|
| `proto/symplex.proto` | filename | `symplex.proto` | Rename to `agent-semantic-protocol.proto` |
| `proto/symplex.proto` | 11 | wrong `go_package` path | Fix to `github.com/olserra/agent-semantic-protocol/proto/gen;asp_proto` |
| `Makefile` | 22 | `proto/symplex.proto` | Update to `proto/agent-semantic-protocol.proto` |

---

## 1. README.md — Complete Rewrite

**Current issues:**

- Wrong `git clone` URL (`agent-semantic-protocol-protocol` is duplicated)
- Wrong import paths in code examples
- No architecture diagram
- No explicit "who is this for" section
- Missing Go version and codecov badges
- Contributing section is too thin

**Target structure:**

```
1. Badge row (CI, codecov, Go version, license, Go Report Card)
2. Hero tagline — one punchy sentence
3. "MCP connects LLMs to tools. ASP connects agents to agents." — the why
4. Core concepts table (already good, keep it)
5. Comparison table (already good, keep it)
6. ASCII architecture diagram
7. Quickstart — 5 min, copy-paste, validated by CI
8. Code examples (fix module path: github.com/olserra/agent-semantic-protocol)
9. Roadmap summary + link to ROADMAP.md
10. Contributing call-to-action + "good first issue" link
11. Community + support section
12. License
```

**Acceptance:** someone unfamiliar with the project can clone, run `make run-handshake`, and understand what happened in under 5 minutes.

---

## 2. CONTRIBUTING.md — Operational Rewrite

**Current issues:** Too generic. Lacks specific workflows, branch naming, PR criteria.

**Target content:**

- Prerequisites (Go 1.22+, make, git)
- Dev environment setup (exact commands)
- How to find an issue (label guide: `good first issue`, `help wanted`, etc.)
- Branch naming: `feat/<name>`, `fix/<issue-number>-<name>`, `docs/<name>`
- Conventional commits with concrete examples from this codebase
- How to run tests (`make test`), lint (`make lint`), examples (`make run-handshake`)
- PR checklist: tests pass, lint clean, description explains *why* not just *what*
- What makes a PR get merged vs rejected
- How wire-format changes are handled (backward compat, spec update required)

---

## 3. CODE_OF_CONDUCT.md

Standard Contributor Covenant v2.1. Signals maturity and professionalism.
Required by GitHub's community health checklist.

---

## 4. SECURITY.md

- Supported versions table
- How to report a vulnerability (private — GitHub private advisory or email)
- Response time commitment (acknowledge within 48h, patch within 14 days for critical)
- Out-of-scope items
- Known security properties (Ed25519 signing, Noise protocol encryption, DID binding)

---

## 5. ROADMAP.md (root level, public-facing)

Different from `docs/plans/` (which is implementation detail).
This is for contributors and users: where is the project going?

**Structure:**

```
## Done (v0.1)
## In Progress (v0.2)
## Planned (v0.3 – v1.0)
## Call for contributors (what we need help with right now)
```

Each item includes: description, effort estimate (S/M/L), and relevant labels.

---

## 6. docs/architecture.md

Technical deep-dive for contributors who want to understand the system before touching code.

**Sections:**

- System overview (the 5 layers: identity, transport, discovery, negotiation, workflow)
- Layer diagram (ASCII)
- Data flow: intent lifecycle from creation to workflow execution
- Key data structures and where they live
- Module dependency graph
- Extension points (where to add new transports, new capability matchers, etc.)

---

## 7. docs/how-it-works.md

Conceptual walkthrough for a broader audience (not just Go developers).
Think "explainer" — how does semantic negotiation actually work?

**Sections:**

- The problem (why existing protocols fall short for agent meshes)
- Intent vectors: what are they and why they work
- The negotiation flow (step-by-step with diagrams)
- Trust and identity (DIDs, Ed25519, trust delta)
- Distributed workflows (how steps compose across peers)
- FAQ

---

## 8. docs/decisions/ — Architecture Decision Records

Three ADRs documenting key choices:

| # | Title | Decision |
|---|-------|----------|
| ADR-001 | Wire format: Protobuf without codegen | Use `protowire` directly to avoid build-time dependencies |
| ADR-002 | Transport: libp2p | Multi-transport (TCP/QUIC/WebRTC), built-in Noise encryption, peer routing |
| ADR-003 | Identity: Ed25519 DIDs | Self-sovereign, fast keygen, compact signatures, compatible with W3C DID spec |

Format: status (Accepted), context, decision, consequences, alternatives considered.

---

## 9. .github/ISSUE_TEMPLATE/

Three templates using GitHub's YAML-based issue form syntax:

**bug_report.yml**

- Go version, OS, reproduction steps, expected vs actual, logs

**feature_request.yml**

- Motivation, use case, proposed API/behavior, alternatives considered, effort willingness

**config.yml**

- Disable blank issues, link to discussions for questions

---

## 10. .github/PULL_REQUEST_TEMPLATE.md

Checklist-based:

- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] Wire format unchanged or spec updated
- [ ] Added/updated tests for new behavior
- [ ] Description explains *why*, not just *what*
- [ ] Linked to relevant issue

---

## 11. .github/workflows/ci.yml — Improvements

**Current:** Go 1.22 and 1.23 matrix only.

**Additions:**

- Add Go 1.24 to matrix (matches go.mod)
- Add `govulncheck` step (Go vulnerability scanner — official Google tool)
- Add "examples" job that runs `make run-handshake` to validate quickstart
- Use latest stable golangci-lint setup-go version

---

## Implementation Order

| Priority | Task | Effort |
|----------|------|--------|
| 1 | Fix Symplex leftovers (Makefile + proto rename) | XS |
| 2 | README.md rewrite | M |
| 3 | CONTRIBUTING.md rewrite | S |
| 4 | Issue templates | S |
| 5 | PR template | XS |
| 6 | CODE_OF_CONDUCT.md | XS |
| 7 | SECURITY.md | S |
| 8 | ROADMAP.md | S |
| 9 | docs/architecture.md | M |
| 10 | docs/how-it-works.md | M |
| 11 | docs/decisions/ ADRs | S |
| 12 | CI improvements | S |

---

## Definition of Done

- [ ] `github.com/olserra/agent-semantic-protocol` community health score: 100%
- [ ] A developer with no prior knowledge can clone → run → understand in < 5 min
- [ ] At least 3 labeled `good first issue` issues exist
- [ ] CI validates the quickstart (examples job)
- [ ] No remaining "symplex" references in source files
