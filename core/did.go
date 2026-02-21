package core

// did.go — Lightweight Decentralized Identifiers for Symplex agents.
//
// Format:  did:symplex:<hex(sha256(ed25519-pubkey))>
//
// The DID is derived deterministically from an Ed25519 public key.
// In v0.1, trust is established by verifying the public key in the
// HandshakeMessage matches the DID prefix.  Message signing (to prove
// key ownership at runtime) is scheduled for v0.2.

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// DID represents a Symplex Decentralized Identifier.
type DID struct {
	Method string // always "symplex" for Symplex DIDs
	ID     string // hex(sha256(pubkey))

	pubKey  []byte
	privKey []byte
}

// NewDID generates a fresh Ed25519 key-pair and derives a DID from it.
func NewDID() (*DID, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("did: key generation failed: %w", err)
	}
	return didFromKey(pub, priv), nil
}

// DIDFromPublicKey derives a DID from a raw Ed25519 public key (no private key).
// Use this when you only know a remote peer's public key.
func DIDFromPublicKey(pubKey []byte) (*DID, error) {
	if len(pubKey) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("did: expected %d-byte public key, got %d",
			ed25519.PublicKeySize, len(pubKey))
	}
	return didFromKey(ed25519.PublicKey(pubKey), nil), nil
}

// ParseDID parses a "did:symplex:<id>" string.
func ParseDID(s string) (*DID, error) {
	var method, id string
	if _, err := fmt.Sscanf(s, "did:%s", &method); err != nil {
		return nil, fmt.Errorf("did: invalid format %q", s)
	}
	// manual parse to avoid sscanf stopping at ':'
	const prefix = "did:"
	if len(s) <= len(prefix) {
		return nil, fmt.Errorf("did: too short %q", s)
	}
	rest := s[len(prefix):]
	for i, c := range rest {
		if c == ':' {
			method = rest[:i]
			id = rest[i+1:]
			break
		}
	}
	if method == "" || id == "" {
		return nil, fmt.Errorf("did: invalid format %q", s)
	}
	return &DID{Method: method, ID: id}, nil
}

func didFromKey(pub ed25519.PublicKey, priv ed25519.PrivateKey) *DID {
	h := sha256.Sum256(pub)
	return &DID{
		Method:  "symplex",
		ID:      hex.EncodeToString(h[:]),
		pubKey:  []byte(pub),
		privKey: []byte(priv),
	}
}

// String returns the canonical DID string ("did:symplex:<id>").
func (d *DID) String() string {
	return fmt.Sprintf("did:%s:%s", d.Method, d.ID)
}

// PublicKey returns a copy of the raw Ed25519 public key (32 bytes), if available.
func (d *DID) PublicKey() []byte {
	if d.pubKey == nil {
		return nil
	}
	out := make([]byte, len(d.pubKey))
	copy(out, d.pubKey)
	return out
}

// Sign signs data with the DID's private key.
// Returns ErrNoPrivateKey if only the public half is available.
func (d *DID) Sign(data []byte) ([]byte, error) {
	if d.privKey == nil {
		return nil, ErrNoPrivateKey
	}
	sig := ed25519.Sign(ed25519.PrivateKey(d.privKey), data)
	return sig, nil
}

// Verify checks that sig is a valid Ed25519 signature of data made with the
// key embedded in this DID.
func (d *DID) Verify(data, sig []byte) bool {
	if d.pubKey == nil {
		return false
	}
	return ed25519.Verify(ed25519.PublicKey(d.pubKey), data, sig)
}

// ValidateBinding confirms that a raw public key matches this DID's embedded ID.
// Call this after receiving a HandshakeMessage to ensure the peer's DID is genuine.
func (d *DID) ValidateBinding(pubKey []byte) bool {
	h := sha256.Sum256(pubKey)
	return hex.EncodeToString(h[:]) == d.ID
}

// ErrNoPrivateKey is returned when signing is attempted without a private key.
var ErrNoPrivateKey = fmt.Errorf("did: private key not available")

// ------------------------------------------------------------------ trust graph

// TrustGraph stores peer-to-peer trust scores in memory.
// It is concurrency-safe — lock before read/write.
type TrustGraph struct {
	scores map[string]float32 // key: "did:symplex:<from>-><to>"
}

// NewTrustGraph creates an empty TrustGraph.
func NewTrustGraph() *TrustGraph {
	return &TrustGraph{scores: make(map[string]float32)}
}

// Set stores the trust score that `from` assigns to `to`.
func (tg *TrustGraph) Set(from, to string, score float32) {
	tg.scores[trustKey(from, to)] = clamp(score)
}

// Get returns the trust score that `from` has assigned to `to`.
// Returns 0 if no entry exists.
func (tg *TrustGraph) Get(from, to string) float32 {
	return tg.scores[trustKey(from, to)]
}

// Apply adds delta to the existing score (clamped to [0,1]).
func (tg *TrustGraph) Apply(from, to string, delta float32) {
	k := trustKey(from, to)
	tg.scores[k] = clamp(tg.scores[k] + delta)
}

func trustKey(from, to string) string { return from + "->" + to }

func clamp(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
