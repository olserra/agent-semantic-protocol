package core

// negotiation.go — Agent Semantic Protocol spontaneous intent negotiation.
//
// Intent negotiation allows Agent A to express a semantic goal (as a float32
// vector) and request that Agent B fulfil it, optionally generating a
// distributed workflow.  Compatibility is measured by cosine similarity
// between the intent vector and each capability's semantic weight vector
// maintained by the receiving agent.

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"crypto/rand"
	"encoding/hex"
)

// NegotiationHandler is a callback invoked when an agent receives an intent.
// Return (response, nil) to accept or reject; return (nil, err) on failure.
type NegotiationHandler func(intent *IntentMessage) (*NegotiationResponse, error)

// DefaultNegotiationHandler builds a NegotiationHandler that accepts any
// intent whose required capabilities are all present in provided.
func DefaultNegotiationHandler(agent *Agent) NegotiationHandler {
	return func(intent *IntentMessage) (*NegotiationResponse, error) {
		missing := missingCapabilities(intent.Capabilities, agent.Capabilities)
		accepted := len(missing) == 0

		reason := "all capabilities available"
		if !accepted {
			reason = fmt.Sprintf("missing capabilities: %v", missing)
		}

		steps := []string{}
		if accepted {
			steps = buildWorkflow(intent)
		}

		resp := &NegotiationResponse{
			RequestID:      intent.ID,
			AgentID:        agent.ID,
			Accepted:       accepted,
			WorkflowSteps:  steps,
			DID:            agent.DID.String(),
			ResponseVector: reflectVector(intent.IntentVector),
			Timestamp:      time.Now().UnixNano(),
			Reason:         reason,
			TrustDelta:     trustDelta(accepted),
		}
		if sig, err := agent.DID.Sign([]byte(resp.RequestID + resp.Reason)); err == nil {
			resp.Signature = sig
		}
		return resp, nil
	}
}

// CreateIntent constructs an IntentMessage ready to be sent.
func CreateIntent(
	sender *Agent,
	intentVector []float32,
	requiredCapabilities []string,
	payload string,
) (*IntentMessage, error) {
	id, err := randomID()
	if err != nil {
		return nil, err
	}
	intent := &IntentMessage{
		ID:           id,
		IntentVector: intentVector,
		Capabilities: requiredCapabilities,
		DID:          sender.DID.String(),
		Payload:      payload,
		Timestamp:    time.Now().UnixNano(),
		TrustScore:   0.5,
		Metadata:     map[string]string{"protocol": ProtocolVersion},
	}
	sig, err := sender.DID.Sign([]byte(intent.ID + intent.Payload))
	if err != nil {
		return nil, fmt.Errorf("CreateIntent: sign: %w", err)
	}
	intent.Signature = sig
	return intent, nil
}

// CosineSimilarity returns the cosine similarity of two equal-length vectors.
// Returns 0 if either vector is zero-length or their lengths differ.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dot, normA, normB float64
	for i := range a {
		dot += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dot / (math.Sqrt(normA) * math.Sqrt(normB))
}

// RankCandidates sorts a list of agents by cosine similarity to the intent
// vector, highest first.  Agents without a registered embedding vector are
// ranked last.
func RankCandidates(intentVector []float32, candidates []AgentProfile) []AgentProfile {
	type ranked struct {
		profile AgentProfile
		score   float64
	}
	rs := make([]ranked, len(candidates))
	for i, c := range candidates {
		rs[i] = ranked{c, CosineSimilarity(intentVector, c.EmbeddingVector)}
	}
	sort.Slice(rs, func(i, j int) bool { return rs[i].score > rs[j].score })
	out := make([]AgentProfile, len(rs))
	for i, r := range rs {
		out[i] = r.profile
	}
	return out
}

// AgentProfile holds a peer agent's public capability profile for ranking.
type AgentProfile struct {
	AgentID         string
	DID             string
	Capabilities    []string
	EmbeddingVector []float32 // Optional representative vector for the agent
	PublicKey       []byte    // Ed25519 public key; set after a handshake
}

// VerifyIntentSignature returns true if intent.Signature is a valid Ed25519
// signature of (intent.ID + intent.Payload) by the owner of pubKey.
// Returns true when Signature is empty (unsigned messages are accepted).
func VerifyIntentSignature(intent *IntentMessage, pubKey []byte) bool {
	if len(intent.Signature) == 0 {
		return true
	}
	d, err := DIDFromPublicKey(pubKey)
	if err != nil {
		return false
	}
	return d.Verify([]byte(intent.ID+intent.Payload), intent.Signature)
}

// VerifyResponseSignature returns true if resp.Signature is a valid Ed25519
// signature of (resp.RequestID + resp.Reason) by the owner of pubKey.
// Returns true when Signature is empty (unsigned messages are accepted).
func VerifyResponseSignature(resp *NegotiationResponse, pubKey []byte) bool {
	if len(resp.Signature) == 0 {
		return true
	}
	d, err := DIDFromPublicKey(pubKey)
	if err != nil {
		return false
	}
	return d.Verify([]byte(resp.RequestID+resp.Reason), resp.Signature)
}

// ------------------------------------------------------------------ in-process negotiation bus

// NegotiationBus enables in-process agents to negotiate without a real network,
// suitable for tests and examples.
type NegotiationBus struct {
	mu       sync.RWMutex
	handlers map[string]NegotiationHandler // keyed by agentID
}

// NewNegotiationBus creates an empty NegotiationBus.
func NewNegotiationBus() *NegotiationBus {
	return &NegotiationBus{handlers: make(map[string]NegotiationHandler)}
}

// Register attaches a handler for the given agentID.
func (b *NegotiationBus) Register(agentID string, h NegotiationHandler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[agentID] = h
}

// Negotiate sends an intent to targetAgentID and returns the response.
func (b *NegotiationBus) Negotiate(targetAgentID string, intent *IntentMessage) (*NegotiationResponse, error) {
	b.mu.RLock()
	h, ok := b.handlers[targetAgentID]
	b.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("negotiation: no handler for agent %q", targetAgentID)
	}
	return h(intent)
}

// ------------------------------------------------------------------ helpers

func missingCapabilities(required, available []string) []string {
	have := make(map[string]struct{}, len(available))
	for _, c := range available {
		have[c] = struct{}{}
	}
	var missing []string
	for _, c := range required {
		if _, ok := have[c]; !ok {
			missing = append(missing, c)
		}
	}
	return missing
}

// buildWorkflow generates a simple deterministic workflow from an intent.
func buildWorkflow(intent *IntentMessage) []string {
	steps := []string{
		fmt.Sprintf("parse_intent:%s", intent.ID),
	}
	for _, cap := range intent.Capabilities {
		steps = append(steps, fmt.Sprintf("execute:%s", cap))
	}
	steps = append(steps, fmt.Sprintf("return_result:%s", intent.ID))
	return steps
}

// reflectVector returns the negation of v (a minimal "response" vector for demos).
func reflectVector(v []float32) []float32 {
	if len(v) == 0 {
		return nil
	}
	out := make([]float32, len(v))
	for i, x := range v {
		out[i] = -x
	}
	return out
}

func trustDelta(accepted bool) float32 {
	if accepted {
		return 0.05
	}
	return -0.02
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("randomID: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// LogIntentMessage logs an intent message.
func LogIntentMessage(intent *IntentMessage) error {
	if intent.Logger != nil {
		logErr := intent.Logger.LogMessage(intent.ID, "IntentMessage", fmt.Sprintf("Capabilities: %v, Payload: %s", intent.Capabilities, intent.Payload))
		if logErr != nil {
			return fmt.Errorf("failed to log intent message: %w", logErr)
		}
	}
	return nil
}
