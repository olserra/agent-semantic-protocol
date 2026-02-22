package core

// encoding.go — Symplex binary encoding using the Protobuf wire format.
//
// Message bytes are produced/consumed with google.golang.org/protobuf/encoding/protowire,
// which gives us the efficient Protobuf binary layout without requiring protoc code generation.
// Field numbers match proto/symplex.proto exactly so generated bindings are compatible.

import (
	"encoding/binary"
	"fmt"
	"math"

	"google.golang.org/protobuf/encoding/protowire"
)

// ------------------------------------------------------------------ encoder

type enc struct{ buf []byte }

func (e *enc) str(field protowire.Number, s string) {
	if s == "" {
		return
	}
	e.buf = protowire.AppendTag(e.buf, field, protowire.BytesType)
	e.buf = protowire.AppendString(e.buf, s)
}

func (e *enc) bytes(field protowire.Number, b []byte) {
	if len(b) == 0 {
		return
	}
	e.buf = protowire.AppendTag(e.buf, field, protowire.BytesType)
	e.buf = protowire.AppendBytes(e.buf, b)
}

func (e *enc) i64(field protowire.Number, v int64) {
	if v == 0 {
		return
	}
	e.buf = protowire.AppendTag(e.buf, field, protowire.VarintType)
	e.buf = protowire.AppendVarint(e.buf, uint64(v))
}

func (e *enc) f32(field protowire.Number, v float32) {
	if v == 0 {
		return
	}
	e.buf = protowire.AppendTag(e.buf, field, protowire.Fixed32Type)
	e.buf = protowire.AppendFixed32(e.buf, math.Float32bits(v))
}

func (e *enc) boolean(field protowire.Number, v bool) {
	if !v {
		return
	}
	e.buf = protowire.AppendTag(e.buf, field, protowire.VarintType)
	e.buf = protowire.AppendVarint(e.buf, 1)
}

func (e *enc) strs(field protowire.Number, ss []string) {
	for _, s := range ss {
		e.buf = protowire.AppendTag(e.buf, field, protowire.BytesType)
		e.buf = protowire.AppendString(e.buf, s)
	}
}

// packedF32 encodes a slice of float32 as a proto3 packed repeated float field.
func (e *enc) packedF32(field protowire.Number, fs []float32) {
	if len(fs) == 0 {
		return
	}
	packed := make([]byte, 0, len(fs)*4)
	for _, f := range fs {
		packed = binary.LittleEndian.AppendUint32(packed, math.Float32bits(f))
	}
	e.buf = protowire.AppendTag(e.buf, field, protowire.BytesType)
	e.buf = protowire.AppendBytes(e.buf, packed)
}

// strMap encodes a map[string]string as proto3 map entries.
// Each entry is a nested message: field 1 = key, field 2 = value.
func (e *enc) strMap(field protowire.Number, m map[string]string) {
	for k, v := range m {
		var entry []byte
		entry = protowire.AppendTag(entry, 1, protowire.BytesType)
		entry = protowire.AppendString(entry, k)
		entry = protowire.AppendTag(entry, 2, protowire.BytesType)
		entry = protowire.AppendString(entry, v)
		e.buf = protowire.AppendTag(e.buf, field, protowire.BytesType)
		e.buf = protowire.AppendBytes(e.buf, entry)
	}
}

// ------------------------------------------------------------------ helpers

func decodePackedF32(packed []byte) []float32 {
	var out []float32
	for len(packed) >= 4 {
		out = append(out, math.Float32frombits(binary.LittleEndian.Uint32(packed[:4])))
		packed = packed[4:]
	}
	return out
}

func decodeStrMapEntry(b []byte) (key, val string, err error) {
	for len(b) > 0 {
		num, _, n := protowire.ConsumeTag(b)
		if n < 0 {
			return "", "", fmt.Errorf("invalid map entry tag")
		}
		b = b[n:]
		switch num {
		case 1:
			s, n2 := protowire.ConsumeString(b)
			if n2 < 0 {
				return "", "", fmt.Errorf("invalid map key")
			}
			key = s
			b = b[n2:]
		case 2:
			s, n2 := protowire.ConsumeString(b)
			if n2 < 0 {
				return "", "", fmt.Errorf("invalid map value")
			}
			val = s
			b = b[n2:]
		default:
			n2 := protowire.ConsumeFieldValue(num, protowire.BytesType, b)
			if n2 < 0 {
				return "", "", fmt.Errorf("invalid map entry field")
			}
			b = b[n2:]
		}
	}
	return key, val, nil
}

// ------------------------------------------------------------------ IntentMessage

// Encode serialises m into the Protobuf wire format.
func (m *IntentMessage) Encode() ([]byte, error) {
	e := &enc{}
	e.str(1, m.ID)
	e.packedF32(2, m.IntentVector)
	e.strs(3, m.Capabilities)
	e.str(4, m.DID)
	e.str(5, m.Payload)
	e.i64(6, m.Timestamp)
	e.f32(7, m.TrustScore)
	e.strMap(8, m.Metadata)
	return e.buf, nil
}

// DecodeIntentMessage deserialises an IntentMessage from wire bytes.
func DecodeIntentMessage(data []byte) (*IntentMessage, error) {
	m := &IntentMessage{Metadata: make(map[string]string)}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("intent: invalid tag")
		}
		data = data[n:]

		switch num {
		case 1:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid id")
			}
			m.ID = s
			data = data[n2:]
		case 2:
			b, n2 := protowire.ConsumeBytes(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid intent_vector")
			}
			m.IntentVector = decodePackedF32(b)
			data = data[n2:]
		case 3:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid capability")
			}
			m.Capabilities = append(m.Capabilities, s)
			data = data[n2:]
		case 4:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid did")
			}
			m.DID = s
			data = data[n2:]
		case 5:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid payload")
			}
			m.Payload = s
			data = data[n2:]
		case 6:
			v, n2 := protowire.ConsumeVarint(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid timestamp")
			}
			m.Timestamp = int64(v)
			data = data[n2:]
		case 7:
			v, n2 := protowire.ConsumeFixed32(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid trust_score")
			}
			m.TrustScore = math.Float32frombits(v)
			data = data[n2:]
		case 8:
			b, n2 := protowire.ConsumeBytes(data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: invalid metadata entry")
			}
			k, v, err := decodeStrMapEntry(b)
			if err != nil {
				return nil, err
			}
			m.Metadata[k] = v
			data = data[n2:]
		default:
			n2 := protowire.ConsumeFieldValue(num, typ, data)
			if n2 < 0 {
				return nil, fmt.Errorf("intent: unknown field %d", num)
			}
			data = data[n2:]
		}
	}
	return m, nil
}

// ------------------------------------------------------------------ HandshakeMessage

// Encode serialises m into the Protobuf wire format.
func (m *HandshakeMessage) Encode() ([]byte, error) {
	e := &enc{}
	e.str(1, m.AgentID)
	e.str(2, m.DID)
	e.strs(3, m.Capabilities)
	e.str(4, m.Version)
	e.i64(5, m.Timestamp)
	e.bytes(6, m.PublicKey)
	e.bytes(7, m.Challenge)
	e.bytes(8, m.ChallengeResponse)
	return e.buf, nil
}

// DecodeHandshakeMessage deserialises a HandshakeMessage from wire bytes.
func DecodeHandshakeMessage(data []byte) (*HandshakeMessage, error) {
	m := &HandshakeMessage{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("handshake: invalid tag")
		}
		data = data[n:]

		switch num {
		case 1:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid agent_id")
			}
			m.AgentID = s
			data = data[n2:]
		case 2:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid did")
			}
			m.DID = s
			data = data[n2:]
		case 3:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid capability")
			}
			m.Capabilities = append(m.Capabilities, s)
			data = data[n2:]
		case 4:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid version")
			}
			m.Version = s
			data = data[n2:]
		case 5:
			v, n2 := protowire.ConsumeVarint(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid timestamp")
			}
			m.Timestamp = int64(v)
			data = data[n2:]
		case 6:
			b, n2 := protowire.ConsumeBytes(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid public_key")
			}
			m.PublicKey = append([]byte(nil), b...)
			data = data[n2:]
		case 7:
			b, n2 := protowire.ConsumeBytes(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid challenge")
			}
			m.Challenge = append([]byte(nil), b...)
			data = data[n2:]
		case 8:
			b, n2 := protowire.ConsumeBytes(data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: invalid challenge_response")
			}
			m.ChallengeResponse = append([]byte(nil), b...)
			data = data[n2:]
		default:
			n2 := protowire.ConsumeFieldValue(num, typ, data)
			if n2 < 0 {
				return nil, fmt.Errorf("handshake: unknown field %d", num)
			}
			data = data[n2:]
		}
	}
	return m, nil
}

// ------------------------------------------------------------------ NegotiationResponse

// Encode serialises m into the Protobuf wire format.
func (m *NegotiationResponse) Encode() ([]byte, error) {
	e := &enc{}
	e.str(1, m.RequestID)
	e.str(2, m.AgentID)
	e.boolean(3, m.Accepted)
	e.strs(4, m.WorkflowSteps)
	e.str(5, m.DID)
	e.packedF32(6, m.ResponseVector)
	e.i64(7, m.Timestamp)
	e.str(8, m.Reason)
	e.f32(9, m.TrustDelta)
	return e.buf, nil
}

// DecodeNegotiationResponse deserialises a NegotiationResponse from wire bytes.
func DecodeNegotiationResponse(data []byte) (*NegotiationResponse, error) {
	m := &NegotiationResponse{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("negoresp: invalid tag")
		}
		data = data[n:]

		switch num {
		case 1:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid request_id")
			}
			m.RequestID = s
			data = data[n2:]
		case 2:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid agent_id")
			}
			m.AgentID = s
			data = data[n2:]
		case 3:
			v, n2 := protowire.ConsumeVarint(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid accepted")
			}
			m.Accepted = v != 0
			data = data[n2:]
		case 4:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid workflow_step")
			}
			m.WorkflowSteps = append(m.WorkflowSteps, s)
			data = data[n2:]
		case 5:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid did")
			}
			m.DID = s
			data = data[n2:]
		case 6:
			b, n2 := protowire.ConsumeBytes(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid response_vector")
			}
			m.ResponseVector = decodePackedF32(b)
			data = data[n2:]
		case 7:
			v, n2 := protowire.ConsumeVarint(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid timestamp")
			}
			m.Timestamp = int64(v)
			data = data[n2:]
		case 8:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid reason")
			}
			m.Reason = s
			data = data[n2:]
		case 9:
			v, n2 := protowire.ConsumeFixed32(data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: invalid trust_delta")
			}
			m.TrustDelta = math.Float32frombits(v)
			data = data[n2:]
		default:
			n2 := protowire.ConsumeFieldValue(num, typ, data)
			if n2 < 0 {
				return nil, fmt.Errorf("negoresp: unknown field %d", num)
			}
			data = data[n2:]
		}
	}
	return m, nil
}

// ------------------------------------------------------------------ WorkflowMessage

// Encode serialises m into the Protobuf wire format.
func (m *WorkflowMessage) Encode() ([]byte, error) {
	e := &enc{}
	e.str(1, m.WorkflowID)
	e.str(2, m.StepID)
	e.str(3, m.NextStepID)
	e.str(4, m.AgentID)
	e.str(5, m.DID)
	e.str(6, m.Action)
	e.strMap(7, m.Params)
	e.str(8, m.ResultChan)
	e.i64(9, m.Timestamp)
	return e.buf, nil
}

// ------------------------------------------------------------------ CapabilityAnnouncement

// Encode serialises m into the Protobuf wire format.
func (m *CapabilityAnnouncement) Encode() ([]byte, error) {
	e := &enc{}
	e.str(1, m.AgentID)
	e.str(2, m.DID)
	e.strs(3, m.Capabilities)
	e.i64(4, m.Timestamp)
	e.i64(5, m.TTL)
	return e.buf, nil
}

// DecodeCapabilityAnnouncement deserialises a CapabilityAnnouncement from wire bytes.
func DecodeCapabilityAnnouncement(data []byte) (*CapabilityAnnouncement, error) {
	m := &CapabilityAnnouncement{}
	for len(data) > 0 {
		num, typ, n := protowire.ConsumeTag(data)
		if n < 0 {
			return nil, fmt.Errorf("capability: invalid tag")
		}
		data = data[n:]

		switch num {
		case 1:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("capability: invalid agent_id")
			}
			m.AgentID = s
			data = data[n2:]
		case 2:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("capability: invalid did")
			}
			m.DID = s
			data = data[n2:]
		case 3:
			s, n2 := protowire.ConsumeString(data)
			if n2 < 0 {
				return nil, fmt.Errorf("capability: invalid capability")
			}
			m.Capabilities = append(m.Capabilities, s)
			data = data[n2:]
		case 4:
			v, n2 := protowire.ConsumeVarint(data)
			if n2 < 0 {
				return nil, fmt.Errorf("capability: invalid timestamp")
			}
			m.Timestamp = int64(v)
			data = data[n2:]
		case 5:
			v, n2 := protowire.ConsumeVarint(data)
			if n2 < 0 {
				return nil, fmt.Errorf("capability: invalid ttl")
			}
			m.TTL = int64(v)
			data = data[n2:]
		default:
			n2 := protowire.ConsumeFieldValue(num, typ, data)
			if n2 < 0 {
				return nil, fmt.Errorf("capability: unknown field %d", num)
			}
			data = data[n2:]
		}
	}
	return m, nil
}

// ------------------------------------------------------------------ framing

// Frame wraps encoded message bytes with a 4-byte big-endian length prefix
// and a 1-byte message type, ready to be sent over a stream.
//
// Layout: [4 bytes: uint32 frame length] [1 byte: MessageType] [N bytes: payload]
func Frame(msgType MessageType, payload []byte) []byte {
	total := 1 + len(payload)
	frame := make([]byte, 4+total)
	binary.BigEndian.PutUint32(frame[:4], uint32(total))
	frame[4] = byte(msgType)
	copy(frame[5:], payload)
	return frame
}

// Unframe reads one framed message, returning the type and raw payload.
// The caller must supply at least 5 bytes (4-byte header + type byte).
func Unframe(frame []byte) (MessageType, []byte, error) {
	if len(frame) < 5 {
		return 0, nil, fmt.Errorf("frame too short (%d bytes)", len(frame))
	}
	total := int(binary.BigEndian.Uint32(frame[:4]))
	if len(frame) < 4+total {
		return 0, nil, fmt.Errorf("frame incomplete: need %d bytes, have %d", 4+total, len(frame))
	}
	msgType := MessageType(frame[4])
	payload := frame[5 : 4+total]
	return msgType, payload, nil
}

// Decode dispatches to the appropriate Decode* function based on msgType.
func Decode(msgType MessageType, data []byte) (interface{}, error) {
	switch msgType {
	case MsgHandshake:
		return DecodeHandshakeMessage(data)
	case MsgIntent:
		return DecodeIntentMessage(data)
	case MsgNegotiation:
		return DecodeNegotiationResponse(data)
	default:
		return nil, fmt.Errorf("unknown message type: 0x%02x", msgType)
	}
}
