// Package picoclaw provides an adapter that bridges Symplex agents with
// Picoclaw-compatible AI assistant services.
//
// Picoclaw is assumed to expose a simple HTTP/JSON API for receiving intents
// and returning structured responses.  This adapter translates Symplex
// IntentMessages into Picoclaw requests and maps responses back to
// NegotiationResponses, allowing any Picoclaw-powered AI assistant to
// participate in a Symplex mesh.
//
// Interface contract (replace base URL with your Picoclaw deployment):
//
//	POST /v1/intent
//	  Body: IntentRequest JSON
//	  Response: IntentResponse JSON
//
//	GET  /v1/capabilities
//	  Response: CapabilitiesResponse JSON
package picoclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/olserra/symplex/core"
)

// Client is a Symplex-to-Picoclaw bridge.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	agentID    string
}

// Option configures a Client.
type Option func(*Client)

// WithAPIKey sets the Bearer token sent with every request.
func WithAPIKey(key string) Option {
	return func(c *Client) { c.apiKey = key }
}

// WithTimeout overrides the default HTTP timeout.
func WithTimeout(d time.Duration) Option {
	return func(c *Client) { c.httpClient.Timeout = d }
}

// WithAgentID sets the agent ID reported in NegotiationResponses.
func WithAgentID(id string) Option {
	return func(c *Client) { c.agentID = id }
}

// NewClient creates a Picoclaw client targeting baseURL (e.g. "https://api.picoclaw.io").
func NewClient(baseURL string, opts ...Option) *Client {
	c := &Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 15 * time.Second},
		agentID:    "picoclaw-bridge",
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

// ------------------------------------------------------------------ API types

// IntentRequest is the JSON payload sent to POST /v1/intent.
type IntentRequest struct {
	AgentID      string            `json:"agent_id"`
	DID          string            `json:"did"`
	IntentVector []float32         `json:"intent_vector"`
	Capabilities []string          `json:"capabilities"`
	Payload      string            `json:"payload,omitempty"`
	TrustScore   float32           `json:"trust_score"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// IntentResponse is the JSON body returned by POST /v1/intent.
type IntentResponse struct {
	RequestID     string   `json:"request_id"`
	Accepted      bool     `json:"accepted"`
	Reason        string   `json:"reason,omitempty"`
	WorkflowSteps []string `json:"workflow_steps,omitempty"`
	TrustDelta    float32  `json:"trust_delta,omitempty"`
}

// CapabilitiesResponse is the JSON body returned by GET /v1/capabilities.
type CapabilitiesResponse struct {
	AgentID      string   `json:"agent_id"`
	DID          string   `json:"did"`
	Capabilities []string `json:"capabilities"`
	Version      string   `json:"version"`
}

// ------------------------------------------------------------------ public API

// SendIntent forwards a Symplex IntentMessage to the Picoclaw API and maps
// the response to a NegotiationResponse.
func (c *Client) SendIntent(ctx context.Context, intent *core.IntentMessage) (*core.NegotiationResponse, error) {
	req := IntentRequest{
		AgentID:      c.agentID,
		DID:          intent.DID,
		IntentVector: intent.IntentVector,
		Capabilities: intent.Capabilities,
		Payload:      intent.Payload,
		TrustScore:   intent.TrustScore,
		Metadata:     intent.Metadata,
	}

	var resp IntentResponse
	if err := c.post(ctx, "/v1/intent", req, &resp); err != nil {
		return nil, fmt.Errorf("picoclaw SendIntent: %w", err)
	}

	return &core.NegotiationResponse{
		RequestID:     intent.ID,
		AgentID:       c.agentID,
		Accepted:      resp.Accepted,
		WorkflowSteps: resp.WorkflowSteps,
		DID:           "",
		Timestamp:     time.Now().UnixNano(),
		Reason:        resp.Reason,
		TrustDelta:    resp.TrustDelta,
	}, nil
}

// FetchCapabilities queries the Picoclaw service for its current capability set.
func (c *Client) FetchCapabilities(ctx context.Context) (*CapabilitiesResponse, error) {
	var resp CapabilitiesResponse
	if err := c.get(ctx, "/v1/capabilities", &resp); err != nil {
		return nil, fmt.Errorf("picoclaw FetchCapabilities: %w", err)
	}
	return &resp, nil
}

// AsNegotiationHandler returns a core.NegotiationHandler backed by this client.
// Useful for registering the Picoclaw adapter on a NegotiationBus.
func (c *Client) AsNegotiationHandler() core.NegotiationHandler {
	return func(intent *core.IntentMessage) (*core.NegotiationResponse, error) {
		return c.SendIntent(context.Background(), intent)
	}
}

// ------------------------------------------------------------------ HTTP helpers

func (c *Client) post(ctx context.Context, path string, body, out interface{}) error {
	data, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

func (c *Client) get(ctx context.Context, path string, out interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("build request: %w", err)
	}
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http do: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
