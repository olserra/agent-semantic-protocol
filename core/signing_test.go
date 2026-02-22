package core_test

import (
	"testing"

	"github.com/olserra/symplex/core"
)

func TestIntentMessageSigning(t *testing.T) {
	agent, err := core.NewAgent("signer", []string{"nlp"})
	if err != nil {
		t.Fatal(err)
	}

	intent, err := core.CreateIntent(agent, []float32{0.5, 0.5}, []string{"nlp"}, "hello")
	if err != nil {
		t.Fatal(err)
	}

	if len(intent.Signature) == 0 {
		t.Fatal("CreateIntent should set a non-empty Signature")
	}

	if !agent.DID.Verify([]byte(intent.ID+intent.Payload), intent.Signature) {
		t.Error("Signature failed to verify against sender DID")
	}
}

func TestIntentSignatureRoundTrip(t *testing.T) {
	agent, err := core.NewAgent("signer", []string{"nlp"})
	if err != nil {
		t.Fatal(err)
	}

	intent, err := core.CreateIntent(agent, []float32{0.1, 0.9}, []string{"nlp"}, "payload")
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := intent.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, err := core.DecodeIntentMessage(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if string(decoded.Signature) != string(intent.Signature) {
		t.Error("Signature not preserved across encode/decode round-trip")
	}

	if !agent.DID.Verify([]byte(decoded.ID+decoded.Payload), decoded.Signature) {
		t.Error("Decoded signature failed to verify")
	}
}

func TestNegotiationResponseSigning(t *testing.T) {
	agent, err := core.NewAgent("responder", []string{"code-gen"})
	if err != nil {
		t.Fatal(err)
	}

	intent := &core.IntentMessage{
		ID:           "test-id",
		Capabilities: []string{"code-gen"},
		DID:          agent.DID.String(),
	}

	h := core.DefaultNegotiationHandler(agent)
	resp, err := h(intent)
	if err != nil {
		t.Fatal(err)
	}

	if len(resp.Signature) == 0 {
		t.Fatal("NegotiationResponse should carry a Signature")
	}

	if !agent.DID.Verify([]byte(resp.RequestID+resp.Reason), resp.Signature) {
		t.Error("NegotiationResponse Signature failed to verify")
	}
}

func TestNegotiationResponseSignatureRoundTrip(t *testing.T) {
	agent, err := core.NewAgent("responder", []string{"nlp"})
	if err != nil {
		t.Fatal(err)
	}

	intent := &core.IntentMessage{ID: "round-trip-id", Capabilities: []string{"nlp"}}
	h := core.DefaultNegotiationHandler(agent)
	resp, err := h(intent)
	if err != nil {
		t.Fatal(err)
	}

	encoded, err := resp.Encode()
	if err != nil {
		t.Fatalf("Encode: %v", err)
	}

	decoded, err := core.DecodeNegotiationResponse(encoded)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}

	if string(decoded.Signature) != string(resp.Signature) {
		t.Error("Signature not preserved across encode/decode round-trip")
	}
}
