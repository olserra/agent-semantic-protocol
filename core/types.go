// Package core provides the fundamental types, encoding, and protocol logic
// for the Symplex semantic agent communication protocol.
package core

import "time"

// MessageType identifies the kind of a framed Symplex message.
type MessageType byte

const (
	MsgHandshake   MessageType = 0x01
	MsgIntent      MessageType = 0x02
	MsgNegotiation MessageType = 0x03
	MsgWorkflow    MessageType = 0x04
	MsgCapability  MessageType = 0x05
)

// ProtocolVersion is the current Symplex wire-protocol version.
const ProtocolVersion = "1.0.0"

// Encoder is implemented by every Symplex message type.
type Encoder interface {
	Encode() ([]byte, error)
	MsgType() MessageType
}

// Agent holds an agent's public identity and capability profile.
// Private key material is kept separate (see DID).
type Agent struct {
	ID           string
	DID          *DID
	Capabilities []string
	pubKey       []byte
	privKey      []byte
}

// NewAgent creates an Agent, generating a fresh Ed25519 key-pair and DID.
func NewAgent(id string, capabilities []string) (*Agent, error) {
	d, err := NewDID()
	if err != nil {
		return nil, err
	}
	return &Agent{
		ID:           id,
		DID:          d,
		Capabilities: capabilities,
		pubKey:       d.pubKey,
		privKey:      d.privKey,
	}, nil
}

// PublicKey returns the raw Ed25519 public key bytes.
func (a *Agent) PublicKey() []byte {
	out := make([]byte, len(a.pubKey))
	copy(out, a.pubKey)
	return out
}

// Sign signs data with the agent's private key.
func (a *Agent) Sign(data []byte) ([]byte, error) {
	return a.DID.Sign(data)
}

// ------------------------------------------------------------------ messages

// IntentMessage carries a semantic intent between agents.
type IntentMessage struct {
	ID           string
	IntentVector []float32         // Semantic embedding (e.g. 384-dim sentence-transformer)
	Capabilities []string          // Capabilities required to fulfil this intent
	DID          string            // Sender DID string ("did:symplex:<id>")
	Payload      string            // Optional payload (plain text or JSON)
	Timestamp    int64             // Unix nanoseconds
	TrustScore   float32           // Sender trust score [0.0, 1.0]
	Metadata     map[string]string // Arbitrary extension metadata
}

func (m *IntentMessage) MsgType() MessageType { return MsgIntent }

// HandshakeMessage establishes agent identity and exchanges capabilities.
type HandshakeMessage struct {
	AgentID           string
	DID               string
	Capabilities      []string
	Version           string
	Timestamp         int64
	PublicKey         []byte // Ed25519 public key
	Challenge         []byte // Random nonce sent to peer
	ChallengeResponse []byte // Signature of peer's challenge with own private key
}

func (m *HandshakeMessage) MsgType() MessageType { return MsgHandshake }

// NegotiationResponse answers an IntentMessage.
type NegotiationResponse struct {
	RequestID      string
	AgentID        string
	Accepted       bool
	WorkflowSteps  []string
	DID            string
	ResponseVector []float32
	Timestamp      int64
	Reason         string
	TrustDelta     float32
}

func (m *NegotiationResponse) MsgType() MessageType { return MsgNegotiation }

// WorkflowMessage carries one step of a distributed workflow.
type WorkflowMessage struct {
	WorkflowID string
	StepID     string
	NextStepID string
	AgentID    string
	DID        string
	Action     string
	Params     map[string]string
	ResultChan string
	Timestamp  int64
}

func (m *WorkflowMessage) MsgType() MessageType { return MsgWorkflow }

// CapabilityAnnouncement broadcasts capabilities to nearby peers.
type CapabilityAnnouncement struct {
	AgentID      string
	DID          string
	Capabilities []string
	Timestamp    int64
	TTL          int64 // seconds; 0 = indefinite
}

func (m *CapabilityAnnouncement) MsgType() MessageType { return MsgCapability }

// now returns current time as Unix nanoseconds.
func now() int64 { return time.Now().UnixNano() }
