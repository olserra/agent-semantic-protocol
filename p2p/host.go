// Package p2p provides a libp2p-backed transport layer for the Symplex protocol.
//
// Each Symplex node wraps a libp2p host and registers stream handlers for the
// "/symplex/1.0.0" protocol ID.  Messages are framed using core.Frame/Unframe:
//
//	[4-byte big-endian length] [1-byte MessageType] [N-byte protobuf payload]
package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	libp2p "github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/libp2p/go-libp2p/core/network"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/libp2p/go-libp2p/core/protocol"
	"github.com/olserra/symplex/core"
)

// SymplexProtocol is the libp2p protocol identifier for Symplex v1.
const SymplexProtocol protocol.ID = "/symplex/1.0.0"

// HandshakeCallback is invoked when a peer initiates a handshake.
// Return a HandshakeMessage to respond, or nil to reject.
type HandshakeCallback func(peerID peer.ID, msg *core.HandshakeMessage) *core.HandshakeMessage

// IntentCallback is invoked when a peer sends an IntentMessage.
// Return a NegotiationResponse to reply.
type IntentCallback func(peerID peer.ID, msg *core.IntentMessage) *core.NegotiationResponse

// AgentHost wraps a libp2p host with Symplex protocol logic.
type AgentHost struct {
	h         host.Host
	agent     *core.Agent
	discovery *core.DiscoveryRegistry
	trust     *core.TrustGraph

	onHandshake HandshakeCallback
	onIntent    IntentCallback
	mu          sync.RWMutex

	// known stores capability profiles by peer.ID string for quick lookup.
	known map[string]core.AgentProfile
}

// NewHost creates a new Symplex P2P host listening on an available TCP port.
// The host's identity is derived from the agent's Ed25519 key.
func NewHost(ctx context.Context, agent *core.Agent) (*AgentHost, error) {
	h, err := libp2p.New(
		libp2p.ListenAddrStrings("/ip4/127.0.0.1/tcp/0"),
	)
	if err != nil {
		return nil, fmt.Errorf("p2p: create host: %w", err)
	}

	ah := &AgentHost{
		h:         h,
		agent:     agent,
		discovery: core.NewDiscoveryRegistry(),
		trust:     core.NewTrustGraph(),
		known:     make(map[string]core.AgentProfile),
	}
	h.SetStreamHandler(SymplexProtocol, ah.handleStream)
	return ah, nil
}

// Close shuts down the libp2p host.
func (ah *AgentHost) Close() error { return ah.h.Close() }

// PeerID returns the underlying libp2p peer.ID.
func (ah *AgentHost) PeerID() peer.ID { return ah.h.ID() }

// AddrInfo returns the peer.AddrInfo that peers can use to connect to us.
func (ah *AgentHost) AddrInfo() peer.AddrInfo {
	return peer.AddrInfo{ID: ah.h.ID(), Addrs: ah.h.Addrs()}
}

// Connect establishes a libp2p connection to a peer.
func (ah *AgentHost) Connect(ctx context.Context, info peer.AddrInfo) error {
	return ah.h.Connect(ctx, info)
}

// Discovery returns the agent's local DiscoveryRegistry.
func (ah *AgentHost) Discovery() *core.DiscoveryRegistry { return ah.discovery }

// Trust returns the agent's TrustGraph.
func (ah *AgentHost) Trust() *core.TrustGraph { return ah.trust }

// OnHandshake registers the callback for incoming handshakes.
func (ah *AgentHost) OnHandshake(fn HandshakeCallback) {
	ah.mu.Lock()
	defer ah.mu.Unlock()
	ah.onHandshake = fn
}

// OnIntent registers the callback for incoming intent messages.
func (ah *AgentHost) OnIntent(fn IntentCallback) {
	ah.mu.Lock()
	defer ah.mu.Unlock()
	ah.onIntent = fn
}

// ------------------------------------------------------------------ outgoing messages

// Handshake initiates a Symplex handshake with peerID.
// Returns the peer's HandshakeMessage on success.
func (ah *AgentHost) Handshake(ctx context.Context, peerID peer.ID) (*core.HandshakeMessage, error) {
	stream, err := ah.h.NewStream(ctx, peerID, SymplexProtocol)
	if err != nil {
		return nil, fmt.Errorf("p2p handshake: open stream: %w", err)
	}
	defer stream.Close()

	// Build and send initiator's handshake.
	ours, err := core.StartHandshake(ah.agent)
	if err != nil {
		return nil, err
	}
	if err := writeMsg(stream, ours); err != nil {
		return nil, fmt.Errorf("p2p handshake: send: %w", err)
	}

	// Read peer's response.
	msgType, data, err := readMsg(stream)
	if err != nil {
		return nil, fmt.Errorf("p2p handshake: recv: %w", err)
	}
	if msgType != core.MsgHandshake {
		return nil, fmt.Errorf("p2p handshake: expected MsgHandshake, got 0x%02x", msgType)
	}
	resp, err := core.DecodeHandshakeMessage(data)
	if err != nil {
		return nil, fmt.Errorf("p2p handshake: decode response: %w", err)
	}

	// Verify the peer signed our challenge.
	if len(resp.ChallengeResponse) > 0 {
		if err := core.FinishHandshake(ours.Challenge, resp); err != nil {
			return nil, err
		}
	}

	// Cache the peer's profile for later lookups.
	ah.mu.Lock()
	ah.known[peerID.String()] = core.AgentProfile{
		AgentID:      resp.AgentID,
		DID:          resp.DID,
		Capabilities: append([]string(nil), resp.Capabilities...),
	}
	ah.mu.Unlock()
	ah.discovery.Announce(ah.known[peerID.String()], 0)

	return resp, nil
}

// SendIntent sends an IntentMessage to peerID and waits for a NegotiationResponse.
func (ah *AgentHost) SendIntent(
	ctx context.Context,
	peerID peer.ID,
	intent *core.IntentMessage,
) (*core.NegotiationResponse, error) {
	stream, err := ah.h.NewStream(ctx, peerID, SymplexProtocol)
	if err != nil {
		return nil, fmt.Errorf("p2p intent: open stream: %w", err)
	}
	defer stream.Close()

	if err := writeMsg(stream, intent); err != nil {
		return nil, fmt.Errorf("p2p intent: send: %w", err)
	}

	msgType, data, err := readMsg(stream)
	if err != nil {
		return nil, fmt.Errorf("p2p intent: recv: %w", err)
	}
	if msgType != core.MsgNegotiation {
		return nil, fmt.Errorf("p2p intent: expected MsgNegotiation, got 0x%02x", msgType)
	}
	resp, err := core.DecodeNegotiationResponse(data)
	if err != nil {
		return nil, fmt.Errorf("p2p intent: decode response: %w", err)
	}

	// Update trust graph.
	ah.trust.Apply(ah.agent.DID.String(), resp.DID, resp.TrustDelta)
	return resp, nil
}

// AnnounceCapabilities broadcasts this agent's capabilities to all connected peers.
func (ah *AgentHost) AnnounceCapabilities(ctx context.Context) {
	ann := core.BuildAnnouncement(ah.agent, 300) // 5-minute TTL
	for _, p := range ah.h.Network().Peers() {
		go func(pid peer.ID) {
			stream, err := ah.h.NewStream(ctx, pid, SymplexProtocol)
			if err != nil {
				return
			}
			defer stream.Close()
			_ = writeMsg(stream, ann)
		}(p)
	}
}

// ------------------------------------------------------------------ incoming stream handler

func (ah *AgentHost) handleStream(s network.Stream) {
	defer s.Close()
	_ = s.SetDeadline(time.Now().Add(30 * time.Second))

	msgType, data, err := readMsg(s)
	if err != nil {
		return
	}

	switch msgType {
	case core.MsgHandshake:
		ah.handleIncomingHandshake(s, data)
	case core.MsgIntent:
		ah.handleIncomingIntent(s, data)
	case core.MsgCapability:
		ah.handleIncomingCapability(data)
	}
}

func (ah *AgentHost) handleIncomingHandshake(s network.Stream, data []byte) {
	incoming, err := core.DecodeHandshakeMessage(data)
	if err != nil {
		return
	}

	// Build response using core.RespondHandshake if no custom callback.
	var resp *core.HandshakeMessage

	ah.mu.RLock()
	cb := ah.onHandshake
	ah.mu.RUnlock()

	if cb != nil {
		resp = cb(s.Conn().RemotePeer(), incoming)
	}
	if resp == nil {
		resp, err = core.RespondHandshake(ah.agent, incoming)
		if err != nil {
			return
		}
	}

	_ = writeMsg(s, resp)

	// Cache peer profile.
	ah.mu.Lock()
	ah.known[s.Conn().RemotePeer().String()] = core.AgentProfile{
		AgentID:      incoming.AgentID,
		DID:          incoming.DID,
		Capabilities: append([]string(nil), incoming.Capabilities...),
	}
	ah.mu.Unlock()
	ah.discovery.Announce(ah.known[s.Conn().RemotePeer().String()], 0)
}

func (ah *AgentHost) handleIncomingIntent(s network.Stream, data []byte) {
	intent, err := core.DecodeIntentMessage(data)
	if err != nil {
		return
	}

	ah.mu.RLock()
	cb := ah.onIntent
	ah.mu.RUnlock()

	var resp *core.NegotiationResponse
	if cb != nil {
		resp = cb(s.Conn().RemotePeer(), intent)
	}
	if resp == nil {
		h := core.DefaultNegotiationHandler(ah.agent)
		resp, _ = h(intent)
	}
	if resp == nil {
		return
	}

	_ = writeMsg(s, resp)
	ah.trust.Apply(ah.agent.DID.String(), intent.DID, resp.TrustDelta)
}

func (ah *AgentHost) handleIncomingCapability(data []byte) {
	ann := &core.CapabilityAnnouncement{}
	_ = ann // decode if needed in v0.2; store raw for now
	// TODO: decode and register in discovery
}

// ------------------------------------------------------------------ wire I/O

// writeMsg serialises msg and writes a framed packet to w.
func writeMsg(w io.Writer, msg core.Encoder) error {
	payload, err := msg.Encode()
	if err != nil {
		return err
	}
	frame := core.Frame(msg.MsgType(), payload)
	_, err = w.Write(frame)
	return err
}

// readMsg reads one framed Symplex message from r.
func readMsg(r io.Reader) (core.MessageType, []byte, error) {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return 0, nil, fmt.Errorf("readMsg header: %w", err)
	}
	n := int(binary.BigEndian.Uint32(hdr[:]))
	if n < 1 || n > 4*1024*1024 { // max 4 MiB
		return 0, nil, fmt.Errorf("readMsg: invalid length %d", n)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return 0, nil, fmt.Errorf("readMsg body: %w", err)
	}
	return core.MessageType(body[0]), body[1:], nil
}
