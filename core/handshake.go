package core

// handshake.go — Symplex connection-establishment protocol.
//
// Flow:
//   Initiator                        Responder
//       |                                |
//       |-- HandshakeMessage (+ nonce) ->|
//       |                                |-- verify DID/key binding
//       |<- HandshakeMessage (+ sig)  ---|
//       |-- verify sig ------------------|
//       |         [capabilities exchanged] |

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

const challengeSize = 32 // bytes

// StartHandshake builds the initiator's HandshakeMessage.
// It embeds a random challenge nonce that the responder must sign.
func StartHandshake(agent *Agent) (*HandshakeMessage, error) {
	nonce := make([]byte, challengeSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("handshake: nonce generation: %w", err)
	}
	return &HandshakeMessage{
		AgentID:      agent.ID,
		DID:          agent.DID.String(),
		Capabilities: agent.Capabilities,
		Version:      ProtocolVersion,
		Timestamp:    time.Now().UnixNano(),
		PublicKey:    agent.PublicKey(),
		Challenge:    nonce,
	}, nil
}

// RespondHandshake processes an incoming HandshakeMessage and builds the
// response.  It verifies the sender's DID/key binding and signs the nonce.
func RespondHandshake(responder *Agent, incoming *HandshakeMessage) (*HandshakeMessage, error) {
	// Verify DID binding: the embedded public key must hash to the claimed DID.
	peerDID, err := ParseDID(incoming.DID)
	if err != nil {
		return nil, fmt.Errorf("handshake: peer DID invalid: %w", err)
	}
	if !peerDID.ValidateBinding(incoming.PublicKey) {
		return nil, fmt.Errorf("handshake: DID/key binding mismatch for %s", incoming.AgentID)
	}

	// Sign the peer's challenge with our private key.
	sig, err := responder.Sign(incoming.Challenge)
	if err != nil {
		return nil, fmt.Errorf("handshake: signing challenge: %w", err)
	}

	// Generate our own challenge.
	nonce := make([]byte, challengeSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("handshake: nonce generation: %w", err)
	}

	return &HandshakeMessage{
		AgentID:           responder.ID,
		DID:               responder.DID.String(),
		Capabilities:      responder.Capabilities,
		Version:           ProtocolVersion,
		Timestamp:         time.Now().UnixNano(),
		PublicKey:         responder.PublicKey(),
		Challenge:         nonce,
		ChallengeResponse: sig,
	}, nil
}

// FinishHandshake verifies the responder's signature over our original challenge.
// originalChallenge is the nonce sent in the initiator's HandshakeMessage.
func FinishHandshake(originalChallenge []byte, response *HandshakeMessage) error {
	peerDID, err := ParseDID(response.DID)
	if err != nil {
		return fmt.Errorf("handshake finish: peer DID invalid: %w", err)
	}
	if !peerDID.ValidateBinding(response.PublicKey) {
		return fmt.Errorf("handshake finish: DID/key binding mismatch for %s", response.AgentID)
	}

	// Reconstruct the peer's DID with their public key so we can verify.
	d, err := DIDFromPublicKey(response.PublicKey)
	if err != nil {
		return fmt.Errorf("handshake finish: invalid public key: %w", err)
	}
	if !d.Verify(originalChallenge, response.ChallengeResponse) {
		return fmt.Errorf("handshake finish: challenge signature invalid for %s", response.AgentID)
	}
	return nil
}

// HandshakeResult collects the outcome of a completed handshake.
type HandshakeResult struct {
	PeerAgentID      string
	PeerDID          string
	PeerCapabilities []string
	PeerPublicKey    []byte
	ProtocolVersion  string
	CompletedAt      time.Time
}

// NewHandshakeResult extracts a HandshakeResult from the responder's message
// after FinishHandshake succeeds.
func NewHandshakeResult(resp *HandshakeMessage) HandshakeResult {
	caps := make([]string, len(resp.Capabilities))
	copy(caps, resp.Capabilities)
	return HandshakeResult{
		PeerAgentID:      resp.AgentID,
		PeerDID:          resp.DID,
		PeerCapabilities: caps,
		PeerPublicKey:    append([]byte(nil), resp.PublicKey...),
		ProtocolVersion:  resp.Version,
		CompletedAt:      time.Now(),
	}
}
