// simple-handshake — Two local Symplex agents exchange a cryptographic handshake
// and discover each other's capabilities over a real libp2p TCP connection.
//
// Run:
//
//	go run ./examples/simple-handshake/main.go
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
	fmt.Println("║     Symplex v0.1 — Simple Handshake Demo     ║")
	fmt.Println("╚══════════════════════════════════════════════╝")
	fmt.Println()

	// ── Create Agent Alpha ─────────────────────────────────────────────────────
	alpha, err := core.NewAgent("agent-alpha", []string{
		"nlp", "reasoning", "intent-parsing",
	})
	if err != nil {
		log.Fatalf("create agent-alpha: %v", err)
	}
	hostA, err := p2p.NewHost(ctx, alpha)
	if err != nil {
		log.Fatalf("create host for agent-alpha: %v", err)
	}
	defer hostA.Close()

	fmt.Printf("✓ Agent Alpha started\n")
	fmt.Printf("  Peer ID : %s\n", hostA.PeerID())
	fmt.Printf("  DID     : %s\n", alpha.DID.String())
	fmt.Printf("  Caps    : %v\n\n", alpha.Capabilities)

	// ── Create Agent Beta ──────────────────────────────────────────────────────
	beta, err := core.NewAgent("agent-beta", []string{
		"code-generation", "math", "vector-search", "storage",
	})
	if err != nil {
		log.Fatalf("create agent-beta: %v", err)
	}
	hostB, err := p2p.NewHost(ctx, beta)
	if err != nil {
		log.Fatalf("create host for agent-beta: %v", err)
	}
	defer hostB.Close()

	fmt.Printf("✓ Agent Beta started\n")
	fmt.Printf("  Peer ID : %s\n", hostB.PeerID())
	fmt.Printf("  DID     : %s\n", beta.DID.String())
	fmt.Printf("  Caps    : %v\n\n", beta.Capabilities)

	// Register a custom handshake callback on Beta so it can log the event.
	hostB.OnHandshake(func(peerID peer.ID, msg *core.HandshakeMessage) *core.HandshakeMessage {
		fmt.Printf("[Beta] ← Handshake from %q  caps=%v\n", msg.AgentID, msg.Capabilities)
		return nil // nil → fall back to default RespondHandshake
	})

	// ── Connect Alpha → Beta ──────────────────────────────────────────────────
	if err := hostA.Connect(ctx, hostB.AddrInfo()); err != nil {
		log.Fatalf("connect alpha→beta: %v", err)
	}
	fmt.Println("✓ TCP connection established (alpha→beta)")

	// ── Handshake ─────────────────────────────────────────────────────────────
	fmt.Println("\n── Performing Symplex Handshake ─────────────────────────────────────")
	resp, err := hostA.Handshake(ctx, hostB.PeerID())
	if err != nil {
		log.Fatalf("handshake failed: %v", err)
	}

	fmt.Printf("\n[Alpha] ✓ Handshake complete!\n")
	fmt.Printf("  Peer agent ID  : %s\n", resp.AgentID)
	fmt.Printf("  Peer DID       : %s\n", resp.DID)
	fmt.Printf("  Peer caps      : %v\n", resp.Capabilities)
	fmt.Printf("  Protocol ver.  : %s\n", resp.Version)

	// ── Discovery registry ────────────────────────────────────────────────────
	fmt.Println("\n── Discovery Registry (Alpha's view) ────────────────────────────────")
	for _, profile := range hostA.Discovery().All() {
		fmt.Printf("  • %s  did=%s  caps=%v\n", profile.AgentID, profile.DID, profile.Capabilities)
	}

	// ── Verify Beta has the capabilities Alpha needs ───────────────────────────
	needed := []string{"code-generation", "math"}
	matches := hostA.Discovery().FindByCapability(needed...)
	fmt.Printf("\n  Agents with %v: %d found\n", needed, len(matches))
	for _, m := range matches {
		fmt.Printf("    → %s\n", m.AgentID)
	}

	fmt.Println("\n✓ Demo complete.")
}
