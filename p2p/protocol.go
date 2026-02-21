package p2p

// protocol.go — Higher-level protocol helpers built on top of AgentHost.
//
// WorkflowOrchestrator coordinates multi-step distributed workflows across a
// set of peer agents, executing each step on the agent that best matches the
// step's required capability vector.

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/olserra/symplex/core"
)

// WorkflowOrchestrator dispatches workflow steps to the best-matching peers.
type WorkflowOrchestrator struct {
	host    *AgentHost
	timeout time.Duration
}

// NewOrchestrator creates a WorkflowOrchestrator backed by the given AgentHost.
func NewOrchestrator(host *AgentHost, stepTimeout time.Duration) *WorkflowOrchestrator {
	return &WorkflowOrchestrator{host: host, timeout: stepTimeout}
}

// StepResult carries the outcome of a single workflow step.
type StepResult struct {
	StepID    string
	AgentID   string
	Accepted  bool
	Reason    string
	Timestamp time.Time
}

// RunWorkflow sends one intent per step to the best-capable peer and collects results.
// steps is a slice of (capabilityTag, intentVector, payload) tuples.
func (o *WorkflowOrchestrator) RunWorkflow(
	ctx context.Context,
	workflowID string,
	steps []WorkflowStep,
) ([]StepResult, error) {
	results := make([]StepResult, len(steps))
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	for i, step := range steps {
		wg.Add(1)
		go func(idx int, s WorkflowStep) {
			defer wg.Done()

			r, err := o.executeStep(ctx, workflowID, s)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				if firstErr == nil {
					firstErr = fmt.Errorf("step %q: %w", s.ID, err)
				}
				results[idx] = StepResult{StepID: s.ID, Accepted: false, Reason: err.Error(), Timestamp: time.Now()}
			} else {
				results[idx] = r
			}
		}(i, step)
	}

	wg.Wait()
	return results, firstErr
}

// WorkflowStep describes one step in a distributed workflow.
type WorkflowStep struct {
	ID           string    // Unique step identifier
	Capability   string    // Required capability for this step
	IntentVector []float32 // Semantic vector describing the step's goal
	Payload      string    // Step-specific payload
}

func (o *WorkflowOrchestrator) executeStep(
	ctx context.Context,
	workflowID string,
	step WorkflowStep,
) (StepResult, error) {
	// Find peers with the required capability.
	candidates := o.host.Discovery().FindByCapability(step.Capability)
	if len(candidates) == 0 {
		return StepResult{}, fmt.Errorf("no peer with capability %q", step.Capability)
	}

	// Rank by cosine similarity.
	ranked := core.RankCandidates(step.IntentVector, candidates)
	best := ranked[0]

	// Resolve peer.ID from the known map (best-effort).
	peerID, err := o.resolvePeerID(best.AgentID)
	if err != nil {
		return StepResult{}, err
	}

	// Build and send intent.
	intent, err := core.CreateIntent(o.host.agent, step.IntentVector,
		[]string{step.Capability}, step.Payload)
	if err != nil {
		return StepResult{}, err
	}
	intent.Metadata["workflow_id"] = workflowID
	intent.Metadata["step_id"] = step.ID

	stepCtx, cancel := context.WithTimeout(ctx, o.timeout)
	defer cancel()

	resp, err := o.host.SendIntent(stepCtx, peerID, intent)
	if err != nil {
		return StepResult{}, err
	}

	return StepResult{
		StepID:    step.ID,
		AgentID:   resp.AgentID,
		Accepted:  resp.Accepted,
		Reason:    resp.Reason,
		Timestamp: time.Now(),
	}, nil
}

func (o *WorkflowOrchestrator) resolvePeerID(agentID string) (peer.ID, error) {
	o.host.mu.RLock()
	defer o.host.mu.RUnlock()
	for pid, profile := range o.host.known {
		if profile.AgentID == agentID {
			id, err := peer.Decode(pid)
			if err != nil {
				return "", fmt.Errorf("resolve peerID for %q: %w", agentID, err)
			}
			return id, nil
		}
	}
	return "", fmt.Errorf("peerID not found for agentID %q", agentID)
}

// ------------------------------------------------------------------ convenience

// DiscoverAndHandshake connects to a peer by AddrInfo, performs a handshake,
// and registers the peer in the discovery registry.
func DiscoverAndHandshake(ctx context.Context, h *AgentHost, info peer.AddrInfo) (core.HandshakeResult, error) {
	if err := h.Connect(ctx, info); err != nil {
		return core.HandshakeResult{}, fmt.Errorf("discover: connect: %w", err)
	}
	resp, err := h.Handshake(ctx, info.ID)
	if err != nil {
		return core.HandshakeResult{}, fmt.Errorf("discover: handshake: %w", err)
	}
	return core.NewHandshakeResult(resp), nil
}
