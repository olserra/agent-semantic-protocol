package core_test

import (
	"testing"
	"time"

	"github.com/olserra/symplex/core"
)

// ------------------------------------------------------------------ IntentMessage

func TestIntentMessageRoundTrip(t *testing.T) {
	original := &core.IntentMessage{
		ID:           "test-intent-001",
		IntentVector: []float32{0.1, 0.5, -0.3, 0.9, 0.0, 1.0},
		Capabilities: []string{"nlp", "reasoning", "code-gen"},
		DID:          "did:symplex:abcdef1234567890",
		Payload:      `{"task":"summarise","lang":"en"}`,
		Timestamp:    time.Now().UnixNano(),
		TrustScore:   0.75,
		Metadata:     map[string]string{"source": "unit-test", "priority": "high"},
	}

	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode failed: %v", err)
	}
	if len(encoded) == 0 {
		t.Fatal("encoded bytes are empty")
	}

	decoded, err := core.DecodeIntentMessage(encoded)
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}

	if decoded.ID != original.ID {
		t.Errorf("ID: got %q want %q", decoded.ID, original.ID)
	}
	if len(decoded.IntentVector) != len(original.IntentVector) {
		t.Fatalf("IntentVector length: got %d want %d",
			len(decoded.IntentVector), len(original.IntentVector))
	}
	for i := range original.IntentVector {
		if decoded.IntentVector[i] != original.IntentVector[i] {
			t.Errorf("IntentVector[%d]: got %v want %v", i,
				decoded.IntentVector[i], original.IntentVector[i])
		}
	}
	if len(decoded.Capabilities) != len(original.Capabilities) {
		t.Fatalf("Capabilities length: got %d want %d",
			len(decoded.Capabilities), len(original.Capabilities))
	}
	for i, c := range original.Capabilities {
		if decoded.Capabilities[i] != c {
			t.Errorf("Capabilities[%d]: got %q want %q", i, decoded.Capabilities[i], c)
		}
	}
	if decoded.DID != original.DID {
		t.Errorf("DID: got %q want %q", decoded.DID, original.DID)
	}
	if decoded.Payload != original.Payload {
		t.Errorf("Payload: got %q want %q", decoded.Payload, original.Payload)
	}
	if decoded.Timestamp != original.Timestamp {
		t.Errorf("Timestamp: got %d want %d", decoded.Timestamp, original.Timestamp)
	}
	if decoded.TrustScore != original.TrustScore {
		t.Errorf("TrustScore: got %v want %v", decoded.TrustScore, original.TrustScore)
	}
	if decoded.Metadata["source"] != original.Metadata["source"] {
		t.Errorf("Metadata[source]: got %q want %q",
			decoded.Metadata["source"], original.Metadata["source"])
	}
}

func TestIntentMessageEmpty(t *testing.T) {
	original := &core.IntentMessage{}
	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode empty: %v", err)
	}
	decoded, err := core.DecodeIntentMessage(encoded)
	if err != nil {
		t.Fatalf("Decode empty: %v", err)
	}
	if decoded.ID != "" || len(decoded.IntentVector) != 0 {
		t.Errorf("non-zero fields in decoded empty message")
	}
}

// ------------------------------------------------------------------ HandshakeMessage

func TestHandshakeMessageRoundTrip(t *testing.T) {
	original := &core.HandshakeMessage{
		AgentID:           "agent-alpha",
		DID:               "did:symplex:deadbeef",
		Capabilities:      []string{"nlp", "vector-search"},
		Version:           "1.0.0",
		Timestamp:         1_000_000_000,
		PublicKey:         []byte("fake32bytepublickey0000000000000"),
		Challenge:         []byte("challenge-nonce-32bytes-padding-"),
		ChallengeResponse: []byte("signature-bytes"),
	}

	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := core.DecodeHandshakeMessage(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.AgentID != original.AgentID {
		t.Errorf("AgentID: got %q want %q", decoded.AgentID, original.AgentID)
	}
	if decoded.DID != original.DID {
		t.Errorf("DID: got %q want %q", decoded.DID, original.DID)
	}
	if decoded.Version != original.Version {
		t.Errorf("Version: got %q want %q", decoded.Version, original.Version)
	}
	if string(decoded.PublicKey) != string(original.PublicKey) {
		t.Errorf("PublicKey mismatch")
	}
	if string(decoded.Challenge) != string(original.Challenge) {
		t.Errorf("Challenge mismatch")
	}
	if string(decoded.ChallengeResponse) != string(original.ChallengeResponse) {
		t.Errorf("ChallengeResponse mismatch")
	}
}

// ------------------------------------------------------------------ NegotiationResponse

func TestNegotiationResponseRoundTrip(t *testing.T) {
	original := &core.NegotiationResponse{
		RequestID:      "req-abc",
		AgentID:        "agent-beta",
		Accepted:       true,
		WorkflowSteps:  []string{"step-1:parse", "step-2:execute", "step-3:return"},
		DID:            "did:symplex:cafebabe",
		ResponseVector: []float32{-0.1, -0.5, 0.3},
		Timestamp:      999,
		Reason:         "all capabilities available",
		TrustDelta:     0.05,
	}

	encoded, err := original.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}
	decoded, err := core.DecodeNegotiationResponse(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if decoded.RequestID != original.RequestID {
		t.Errorf("RequestID: got %q want %q", decoded.RequestID, original.RequestID)
	}
	if !decoded.Accepted {
		t.Error("Accepted: got false want true")
	}
	if len(decoded.WorkflowSteps) != len(original.WorkflowSteps) {
		t.Fatalf("WorkflowSteps length: got %d want %d",
			len(decoded.WorkflowSteps), len(original.WorkflowSteps))
	}
	if decoded.TrustDelta != original.TrustDelta {
		t.Errorf("TrustDelta: got %v want %v", decoded.TrustDelta, original.TrustDelta)
	}
}

// ------------------------------------------------------------------ framing

func TestFrameUnframe(t *testing.T) {
	payload := []byte("hello symplex")
	framed := core.Frame(core.MsgIntent, payload)

	msgType, unframed, err := core.Unframe(framed)
	if err != nil {
		t.Fatalf("Unframe: %v", err)
	}
	if msgType != core.MsgIntent {
		t.Errorf("msgType: got %d want %d", msgType, core.MsgIntent)
	}
	if string(unframed) != string(payload) {
		t.Errorf("payload: got %q want %q", unframed, payload)
	}
}

func TestUnframeShort(t *testing.T) {
	_, _, err := core.Unframe([]byte{1, 2})
	if err == nil {
		t.Error("expected error for short frame, got nil")
	}
}

// ------------------------------------------------------------------ DID

func TestDIDGenAndString(t *testing.T) {
	d, err := core.NewDID()
	if err != nil {
		t.Fatalf("NewDID: %v", err)
	}
	s := d.String()
	if len(s) < 10 {
		t.Errorf("DID string too short: %q", s)
	}

	parsed, err := core.ParseDID(s)
	if err != nil {
		t.Fatalf("ParseDID(%q): %v", s, err)
	}
	if parsed.ID != d.ID {
		t.Errorf("parsed ID %q != original %q", parsed.ID, d.ID)
	}
}

func TestDIDBindingValidation(t *testing.T) {
	d, _ := core.NewDID()
	pub := d.PublicKey()

	if !d.ValidateBinding(pub) {
		t.Error("ValidateBinding should return true for own key")
	}

	tampered := make([]byte, len(pub))
	copy(tampered, pub)
	tampered[0] ^= 0xFF
	if d.ValidateBinding(tampered) {
		t.Error("ValidateBinding should return false for tampered key")
	}
}

func TestDIDSignVerify(t *testing.T) {
	d, _ := core.NewDID()
	msg := []byte("symplex test message")

	sig, err := d.Sign(msg)
	if err != nil {
		t.Fatalf("Sign: %v", err)
	}
	if !d.Verify(msg, sig) {
		t.Error("Verify should return true for own signature")
	}

	tampered := append([]byte(nil), msg...)
	tampered[0] ^= 0xFF
	if d.Verify(tampered, sig) {
		t.Error("Verify should return false for tampered message")
	}
}

// ------------------------------------------------------------------ cosine similarity

func TestCosineSimilarity(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{1, 0, 0}
	sim := core.CosineSimilarity(a, b)
	if sim < 0.999 {
		t.Errorf("identical vectors: expected ~1.0, got %f", sim)
	}

	c := []float32{0, 1, 0}
	sim2 := core.CosineSimilarity(a, c)
	if sim2 > 0.001 {
		t.Errorf("orthogonal vectors: expected ~0.0, got %f", sim2)
	}
}

// ------------------------------------------------------------------ discovery

func TestDiscoveryRegistry(t *testing.T) {
	reg := core.NewDiscoveryRegistry()

	reg.Announce(core.AgentProfile{
		AgentID:      "agent-1",
		DID:          "did:symplex:aa",
		Capabilities: []string{"nlp", "reasoning"},
	}, 0)

	reg.Announce(core.AgentProfile{
		AgentID:      "agent-2",
		DID:          "did:symplex:bb",
		Capabilities: []string{"code-gen", "nlp"},
	}, 0)

	results := reg.FindByCapability("nlp")
	if len(results) != 2 {
		t.Errorf("FindByCapability(nlp): expected 2, got %d", len(results))
	}

	results2 := reg.FindByCapability("code-gen")
	if len(results2) != 1 || results2[0].AgentID != "agent-2" {
		t.Errorf("FindByCapability(code-gen): unexpected result %v", results2)
	}

	results3 := reg.FindByCapability("unknown-cap")
	if len(results3) != 0 {
		t.Errorf("FindByCapability(unknown): expected 0, got %d", len(results3))
	}
}
