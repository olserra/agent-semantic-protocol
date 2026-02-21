package core

// discovery.go — Dynamic capability discovery registry.
//
// Agents announce their capabilities via CapabilityAnnouncement messages.
// The registry indexes them for fast lookup, supporting TTL-based expiry.

import (
	"fmt"
	"sync"
	"time"
)

// DiscoveryRegistry stores announced capability profiles from peer agents.
// All methods are concurrency-safe.
type DiscoveryRegistry struct {
	mu      sync.RWMutex
	entries map[string]*registryEntry // keyed by AgentID
}

type registryEntry struct {
	profile   AgentProfile
	expiresAt time.Time // zero value means no expiry
}

// NewDiscoveryRegistry creates an empty registry.
func NewDiscoveryRegistry() *DiscoveryRegistry {
	return &DiscoveryRegistry{entries: make(map[string]*registryEntry)}
}

// Announce registers or updates an agent's capability profile.
// ttlSeconds == 0 means the entry never expires.
func (r *DiscoveryRegistry) Announce(profile AgentProfile, ttlSeconds int64) {
	r.mu.Lock()
	defer r.mu.Unlock()

	var exp time.Time
	if ttlSeconds > 0 {
		exp = time.Now().Add(time.Duration(ttlSeconds) * time.Second)
	}
	r.entries[profile.AgentID] = &registryEntry{profile: profile, expiresAt: exp}
}

// AnnounceFromMessage registers the agent described by a CapabilityAnnouncement.
func (r *DiscoveryRegistry) AnnounceFromMessage(msg *CapabilityAnnouncement) {
	r.Announce(AgentProfile{
		AgentID:      msg.AgentID,
		DID:          msg.DID,
		Capabilities: append([]string(nil), msg.Capabilities...),
	}, msg.TTL)
}

// Remove deletes an agent's entry from the registry.
func (r *DiscoveryRegistry) Remove(agentID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.entries, agentID)
}

// FindByCapability returns all live agents that declare ALL of required capabilities.
func (r *DiscoveryRegistry) FindByCapability(required ...string) []AgentProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var results []AgentProfile
	for _, e := range r.entries {
		if e.isExpired() {
			continue
		}
		if hasAll(e.profile.Capabilities, required) {
			results = append(results, e.profile)
		}
	}
	return results
}

// FindByDID returns the profile registered for a specific DID, or false.
func (r *DiscoveryRegistry) FindByDID(did string) (AgentProfile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.entries {
		if e.isExpired() {
			continue
		}
		if e.profile.DID == did {
			return e.profile, true
		}
	}
	return AgentProfile{}, false
}

// All returns a snapshot of all live profiles.
func (r *DiscoveryRegistry) All() []AgentProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var out []AgentProfile
	for _, e := range r.entries {
		if !e.isExpired() {
			out = append(out, e.profile)
		}
	}
	return out
}

// Evict removes all expired entries and returns the count removed.
func (r *DiscoveryRegistry) Evict() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	n := 0
	for id, e := range r.entries {
		if e.isExpired() {
			delete(r.entries, id)
			n++
		}
	}
	return n
}

// StartEvictionLoop runs a background goroutine that periodically evicts
// expired entries.  Cancel ctx to stop it.
func (r *DiscoveryRegistry) StartEvictionLoop(interval time.Duration, done <-chan struct{}) {
	go func() {
		t := time.NewTicker(interval)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				r.Evict()
			case <-done:
				return
			}
		}
	}()
}

// BuildAnnouncement creates a CapabilityAnnouncement for the given agent.
func BuildAnnouncement(agent *Agent, ttlSeconds int64) *CapabilityAnnouncement {
	caps := make([]string, len(agent.Capabilities))
	copy(caps, agent.Capabilities)
	return &CapabilityAnnouncement{
		AgentID:      agent.ID,
		DID:          agent.DID.String(),
		Capabilities: caps,
		Timestamp:    now(),
		TTL:          ttlSeconds,
	}
}

// CapabilitySetDiff computes which of required are absent from available.
func CapabilitySetDiff(required, available []string) (present, absent []string) {
	have := make(map[string]struct{}, len(available))
	for _, c := range available {
		have[c] = struct{}{}
	}
	for _, c := range required {
		if _, ok := have[c]; ok {
			present = append(present, c)
		} else {
			absent = append(absent, c)
		}
	}
	return
}

// SummariseRegistry returns a human-readable summary for logging.
func SummariseRegistry(r *DiscoveryRegistry) string {
	profiles := r.All()
	return fmt.Sprintf("DiscoveryRegistry: %d live agents", len(profiles))
}

// ------------------------------------------------------------------ helpers

func (e *registryEntry) isExpired() bool {
	if e.expiresAt.IsZero() {
		return false
	}
	return time.Now().After(e.expiresAt)
}

func hasAll(available, required []string) bool {
	have := make(map[string]struct{}, len(available))
	for _, c := range available {
		have[c] = struct{}{}
	}
	for _, r := range required {
		if _, ok := have[r]; !ok {
			return false
		}
	}
	return true
}
