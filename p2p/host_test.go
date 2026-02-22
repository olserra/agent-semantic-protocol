package p2p_test

import (
	"context"
	"testing"
	"time"

	"github.com/olserra/symplex/core"
	"github.com/olserra/symplex/p2p"
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
	h, err := p2p.NewHost(context.Background(), agent)
	if err != nil {
		t.Fatalf("NewHost: %v", err)
	}
	t.Cleanup(func() { _ = h.Close() })
	return h
}

// TestHandshake verifies that two agents complete the Symplex handshake and
// exchange capability information.
func TestHandshake(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp"})
	beta := makeAgent(t, "beta", []string{"code-gen"})

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

// TestHandshakeRegistersInDiscovery verifies that a completed handshake registers
// the remote peer in the local DiscoveryRegistry.
func TestHandshakeRegistersInDiscovery(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp"})
	beta := makeAgent(t, "beta", []string{"code-gen", "reasoning"})

	hA := makeHost(t, alpha)
	hB := makeHost(t, beta)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hA.Connect(ctx, hB.AddrInfo()); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if _, err := hA.Handshake(ctx, hB.PeerID()); err != nil {
		t.Fatalf("Handshake: %v", err)
	}

	found := hA.Discovery().FindByCapability("code-gen")
	if len(found) == 0 {
		t.Error("beta should be discoverable by alpha after handshake")
	}
}

// TestSendIntentAccepted verifies that an intent is accepted when the peer has
// all required capabilities.
func TestSendIntentAccepted(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp"})
	beta := makeAgent(t, "beta", []string{"summarisation"})

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

// TestSendIntentRejected verifies rejection when the peer lacks required capabilities.
func TestSendIntentRejected(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp"})
	beta := makeAgent(t, "beta", []string{"code-gen"}) // does NOT have summarisation

	hA := makeHost(t, alpha)
	hB := makeHost(t, beta)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := hA.Connect(ctx, hB.AddrInfo()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	intent, err := core.CreateIntent(alpha, []float32{0.5, 0.5}, []string{"summarisation"}, "")
	if err != nil {
		t.Fatalf("CreateIntent: %v", err)
	}

	resp, err := hA.SendIntent(ctx, hB.PeerID(), intent)
	if err != nil {
		t.Fatalf("SendIntent: %v", err)
	}

	if resp.Accepted {
		t.Error("expected intent to be rejected, was accepted")
	}
}

// TestAnnounceCapabilities verifies that AnnounceCapabilities registers the
// announcing agent in the receiver's DiscoveryRegistry via MsgCapability.
func TestAnnounceCapabilities(t *testing.T) {
	alpha := makeAgent(t, "alpha", []string{"nlp", "reasoning"})
	beta := makeAgent(t, "beta", []string{"code-gen"})

	hA := makeHost(t, alpha)
	hB := makeHost(t, beta)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// beta connects to alpha so alpha can reach beta when announcing
	if err := hB.Connect(ctx, hA.AddrInfo()); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	hA.AnnounceCapabilities(ctx)
	time.Sleep(300 * time.Millisecond) // allow async streams to complete

	found := hB.Discovery().FindByCapability("nlp")
	if len(found) == 0 {
		t.Error("expected alpha to be discoverable by beta after AnnounceCapabilities")
	}
}
