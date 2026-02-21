// negotiation-demo — Agent A sends a semantic intent to Agent B, which
// evaluates compatibility and responds with a distributed workflow plan.
// The demo exercises the full Symplex negotiation loop over libp2p.
//
// Run:
//
//	go run ./examples/negotiation-demo/main.go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/olserra/symplex/core"
	"github.com/olserra/symplex/p2p"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Println("╔══════════════════════════════════════════════╗")
	fmt.Println("║  Symplex v0.1 — Intent Negotiation Demo      ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	// ── Agents ────────────────────────────────────────────────────────────────
	requester, err := core.NewAgent("requester", []string{"nlp", "intent-routing"})
	if err != nil {
		log.Fatalf("create requester: %v", err)
	}

	codeAgent, err := core.NewAgent("code-agent", []string{
		"code-generation", "python", "typescript", "validation",
	})
	if err != nil {
		log.Fatalf("create code-agent: %v", err)
	}

	limitedAgent, err := core.NewAgent("limited-agent", []string{"storage"})
	if err != nil {
		log.Fatalf("create limited-agent: %v", err)
	}

	// ── Hosts ─────────────────────────────────────────────────────────────────
	hostR, err := p2p.NewHost(ctx, requester)
	if err != nil {
		log.Fatalf("host requester: %v", err)
	}
	defer hostR.Close()

	hostC, err := p2p.NewHost(ctx, codeAgent)
	if err != nil {
		log.Fatalf("host code-agent: %v", err)
	}
	defer hostC.Close()

	hostL, err := p2p.NewHost(ctx, limitedAgent)
	if err != nil {
		log.Fatalf("host limited-agent: %v", err)
	}
	defer hostL.Close()

	// ── Custom intent handler for code-agent ──────────────────────────────────
	hostC.OnIntent(func(peerID peer.ID, intent *core.IntentMessage) *core.NegotiationResponse {
		fmt.Printf("\n[code-agent] ← Intent from %s\n", peerID.ShortString())
		fmt.Printf("             Vector (first 5): %.3f\n", intent.IntentVector[:5])
		fmt.Printf("             Required caps:    %v\n", intent.Capabilities)
		fmt.Printf("             Payload:          %s\n", intent.Payload)

		// Use default handler logic then augment the workflow.
		h := core.DefaultNegotiationHandler(codeAgent)
		resp, _ := h(intent)
		if resp.Accepted {
			// Enrich with a more detailed workflow.
			resp.WorkflowSteps = []string{
				"step-1: parse_natural_language",
				"step-2: generate_code[python]",
				"step-3: run_static_analysis",
				"step-4: validate_output",
				"step-5: stream_result_to_requester",
			}
		}
		return resp
	})

	// ── Connect Requester to both peers ───────────────────────────────────────
	for _, info := range []peer.AddrInfo{hostC.AddrInfo(), hostL.AddrInfo()} {
		if err := hostR.Connect(ctx, info); err != nil {
			log.Fatalf("connect: %v", err)
		}
	}

	// ── Handshake to populate discovery registry ──────────────────────────────
	fmt.Println("── Handshake phase ──────────────────────────────────────────────────")
	for _, pid := range []peer.ID{hostC.PeerID(), hostL.PeerID()} {
		result, err := p2p.DiscoverAndHandshake(ctx, hostR, peer.AddrInfo{
			ID:    pid,
			Addrs: nil,
		})
		if err != nil {
			// Retry with full AddrInfo.
			var info peer.AddrInfo
			if pid == hostC.PeerID() {
				info = hostC.AddrInfo()
			} else {
				info = hostL.AddrInfo()
			}
			result, err = p2p.DiscoverAndHandshake(ctx, hostR, info)
			if err != nil {
				log.Fatalf("handshake: %v", err)
			}
		}
		fmt.Printf("  ✓ Handshook %s  caps=%v\n", result.PeerAgentID, result.PeerCapabilities)
	}

	// ── Build a semantic intent ───────────────────────────────────────────────
	// This vector represents "generate Python code from natural language".
	// In production, derive this from a sentence-transformer model.
	intentVector := []float32{
		0.82, 0.14, 0.76, 0.05, 0.91,
		0.33, 0.67, 0.48, 0.21, 0.88,
	}

	fmt.Println("\n── Negotiation phase ────────────────────────────────────────────────")
	fmt.Println("  Requester sends intent: 'Generate Python from natural language'")
	fmt.Printf("  Intent vector (10-dim): %.2f\n", intentVector)
	fmt.Println()

	// ── Send intent to code-agent ─────────────────────────────────────────────
	intent, err := core.CreateIntent(
		requester,
		intentVector,
		[]string{"code-generation", "python"},
		"Write a Python function to merge two sorted lists into one sorted list.",
	)
	if err != nil {
		log.Fatalf("create intent: %v", err)
	}

	resp, err := hostR.SendIntent(ctx, hostC.PeerID(), intent)
	if err != nil {
		log.Fatalf("send intent to code-agent: %v", err)
	}
	printNegotiationResult("[code-agent]", resp)

	// ── Send same intent to limited-agent (should be rejected) ────────────────
	resp2, err := hostR.SendIntent(ctx, hostL.PeerID(), intent)
	if err != nil {
		log.Fatalf("send intent to limited-agent: %v", err)
	}
	printNegotiationResult("[limited-agent]", resp2)

	// ── Trust graph ───────────────────────────────────────────────────────────
	fmt.Println("\n── Trust Graph (Requester's view) ───────────────────────────────────")
	fmt.Printf("  trust(requester → code-agent)   = %.2f\n",
		hostR.Trust().Get(requester.DID.String(), codeAgent.DID.String()))
	fmt.Printf("  trust(requester → limited-agent) = %.2f\n",
		hostR.Trust().Get(requester.DID.String(), limitedAgent.DID.String()))

	// ── In-process negotiation bus demo (no network required) ────────────────
	fmt.Println("\n── In-Process NegotiationBus Demo (goroutines, no TCP) ──────────────")
	bus := core.NewNegotiationBus()
	bus.Register("code-agent-local", core.DefaultNegotiationHandler(codeAgent))

	localIntent, _ := core.CreateIntent(requester, intentVector,
		[]string{"code-generation"}, "local negotiation test")
	localResp, err := bus.Negotiate("code-agent-local", localIntent)
	if err != nil {
		log.Fatalf("bus negotiate: %v", err)
	}
	fmt.Printf("  Bus negotiation accepted=%v  steps=%v\n",
		localResp.Accepted, localResp.WorkflowSteps)

	fmt.Println("\n✓ Demo complete.")
}

func printNegotiationResult(label string, resp *core.NegotiationResponse) {
	status := "✗ REJECTED"
	if resp.Accepted {
		status = "✓ ACCEPTED"
	}
	fmt.Printf("  %s %s\n", label, status)
	fmt.Printf("    Agent    : %s\n", resp.AgentID)
	fmt.Printf("    Reason   : %s\n", resp.Reason)
	fmt.Printf("    TrustΔ   : %+.2f\n", resp.TrustDelta)
	if resp.Accepted {
		fmt.Printf("    Workflow :\n")
		for i, s := range resp.WorkflowSteps {
			fmt.Printf("      [%d] %s\n", i+1, s)
		}
	}
	fmt.Printf("    Latency  : %s\n",
		time.Unix(0, resp.Timestamp).Format("15:04:05.000"))
	fmt.Println()
}
