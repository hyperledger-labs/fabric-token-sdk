/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package session

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	session "github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/session"
)

// CurrentVersion is the protocol version stamped on every outgoing envelope.
// Receivers reject messages whose version differs from their own CurrentVersion.
const CurrentVersion uint32 = 1

// Envelope wraps all TTX interactive protocol messages with version and type
// information. Wire format uses compact field names for efficiency:
//   - "v" = Version (uint32, monotonic)
//   - "t" = Type discriminator (string, mandatory)
//   - "b" = Body (json.RawMessage, the actual payload)
type Envelope struct {
	Version uint32          `json:"v"`
	Type    string          `json:"t"`
	Body    json.RawMessage `json:"b"`
}

// Validate checks that the envelope carries the expected message type.
func (e *Envelope) Validate(expectedType string) error {
	if e.Type != expectedType {
		return errors.Join(errors.Errorf("expected %s, got %s", expectedType, e.Type), ErrTypeMismatch)
	}

	return nil
}

// Message type constants for all JSON-typed interactive protocol messages.
const (
	// recipients.go
	TypeRecipientRequest         = "recipient_req"
	TypeRecipientResponse        = "recipient_resp"
	TypeExchangeRecipientRequest = "exchange_req"
	TypeExchangeRecipientResp    = "exchange_resp"
	TypeMultisigRecipientData    = "multisig_data"
	TypePolicyRecipientData      = "policy_data"

	// withdrawal.go
	TypeWithdrawalRequest = "withdrawal_req"

	// upgrade.go
	TypeUpgradeAgreement = "upgrade_agree"
	TypeUpgradeRequest   = "upgrade_req"

	// multisig/spend.go and boolpolicy/spend.go
	TypeSpendRequest  = "spend_req"
	TypeSpendResponse = "spend_resp"

	// collectendorsements.go, endorse.go, accept.go, auditor.go, receivetx.go
	TypeSignatureRequest    = "sig_req"
	TypeSignature           = "signature"
	TypeTransaction         = "transaction"
	TypeTransactionResponse = "tx_resp"

	// collectactions.go
	TypeActions        = "actions"
	TypeActionTransfer = "action_transfer"

	// interop/htlc/distribute.go
	TypeHTLCTerms = "htlc_terms"
)

// Sentinel errors for envelope validation.
var (
	ErrVersionMismatch    = errors.New("protocol version mismatch")
	ErrUnsupportedVersion = errors.New("unsupported protocol version")
	ErrMissingVersion     = errors.New("missing protocol version")
	ErrInvalidEnvelope    = errors.New("invalid envelope format")
	ErrTypeMismatch       = errors.New("message type mismatch")
)

// VersionError provides structured detail about a version mismatch.
type VersionError struct {
	Expected uint32
	Received uint32
	Message  string
}

func (e *VersionError) Error() string {
	if e.Message != "" {
		return fmt.Sprintf("protocol version mismatch: expected %d, received %d: %s", e.Expected, e.Received, e.Message)
	}

	return fmt.Sprintf("protocol version mismatch: expected %d, received %d", e.Expected, e.Received)
}

func (e *VersionError) Is(target error) bool {
	return target == ErrVersionMismatch
}

// VersionCompatibility defines which protocol versions can communicate.
// For v1, only same-version communication is supported.
var VersionCompatibility = map[uint32][]uint32{
	1: {1},
}

// IsCompatible returns true if local and remote versions can interoperate.
func IsCompatible(local, remote uint32) bool {
	compatible, ok := VersionCompatibility[local]
	if !ok {
		return false
	}
	for _, v := range compatible {
		if v == remote {
			return true
		}
	}

	return false
}

// WrapEnvelope marshals v into an Envelope with the given message type.
func WrapEnvelope(v any, msgType string) (*Envelope, error) {
	body, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal envelope body")
	}

	return &Envelope{
		Version: CurrentVersion,
		Type:    msgType,
		Body:    body,
	}, nil
}

// UnwrapEnvelope decodes raw bytes into an Envelope and validates version
// and (optionally) message type. If expectedType is non-empty, the type is
// checked; pass "" to skip the type check.
func UnwrapEnvelope(raw []byte, expectedType string) (*Envelope, error) {
	var env Envelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, errors.Join(errors.Wrap(err, "invalid envelope format"), ErrInvalidEnvelope)
	}
	if env.Version == 0 {
		return nil, ErrMissingVersion
	}
	if env.Version != CurrentVersion {
		return nil, &VersionError{Expected: CurrentVersion, Received: env.Version}
	}
	if len(env.Type) == 0 {
		return nil, errors.Join(errors.New("type field is empty"), ErrInvalidEnvelope)
	}
	if expectedType != "" {
		if err := env.Validate(expectedType); err != nil {
			return nil, err
		}
	}

	return &env, nil
}

// UnwrapBody is a convenience that unwraps an envelope and unmarshals the body
// into dst in a single call.
func UnwrapBody(raw []byte, expectedType string, dst any) error {
	env, err := UnwrapEnvelope(raw, expectedType)
	if err != nil {
		return err
	}

	return json.Unmarshal(env.Body, dst)
}

// SendTyped wraps v in a versioned envelope with the given message type and
// sends it over the session.
func SendTyped(s *session.S, ctx context.Context, v any, msgType string) error {
	return SendTypedWithMetrics(s, ctx, v, msgType, nil)
}

// SendTypedWithMetrics is like SendTyped but also records envelope metrics.
func SendTypedWithMetrics(s *session.S, ctx context.Context, v any, msgType string, m *EnvelopeMetrics) error {
	env, err := WrapEnvelope(v, msgType)
	if err != nil {
		return err
	}
	m.observeSend(msgType, len(env.Body))

	return s.SendWithContext(ctx, env)
}

// ReceiveTyped receives a versioned envelope, validates its version and type,
// and unmarshals the body into dst.
func ReceiveTyped(s *session.S, expectedType string, dst any) error {
	return ReceiveTypedWithTimeout(s, expectedType, dst, session.DefaultReceiveTimeout)
}

// ReceiveTypedWithTimeout is like ReceiveTyped but with an explicit timeout.
func ReceiveTypedWithTimeout(s *session.S, expectedType string, dst any, d time.Duration) error {
	return ReceiveTypedWithTimeoutAndMetrics(s, expectedType, dst, d, nil)
}

// ReceiveTypedWithTimeoutAndMetrics is like ReceiveTypedWithTimeout but also
// records envelope metrics.
func ReceiveTypedWithTimeoutAndMetrics(s *session.S, expectedType string, dst any, d time.Duration, m *EnvelopeMetrics) error {
	raw, err := s.ReceiveRawWithTimeout(d)
	if err != nil {
		return err
	}

	env, err := UnwrapEnvelope(raw, expectedType)
	if err != nil {
		m.observeError(classifyError(err))

		return err
	}
	m.observeReceive(env)

	return json.Unmarshal(env.Body, dst)
}

// SendEnvelopeOnSession wraps v in a versioned envelope and sends it on view.Session.
func SendEnvelopeOnSession(sess view.Session, ctx context.Context, v any, msgType string) error {
	env, err := WrapEnvelope(v, msgType)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return errors.Wrap(err, "failed to marshal envelope")
	}

	return sess.SendWithContext(ctx, raw)
}

// ReceiveEnvelopeFromSession receives a versioned envelope from view.Session.
func ReceiveEnvelopeFromSession(sess view.Session, ctx context.Context, expectedType string, d time.Duration) (*Envelope, error) {
	s := session.New(sess, ctx, JSONMarshaller{})

	return receiveEnvelope(s, expectedType, d)
}

func receiveEnvelope(s *session.S, expectedType string, d time.Duration) (*Envelope, error) {
	raw, err := s.ReceiveRawWithTimeout(d)
	if err != nil {
		return nil, err
	}

	return UnwrapEnvelope(raw, expectedType)
}

func classifyError(err error) string {
	switch {
	case errors.Is(err, ErrMissingVersion):
		return "missing_version"
	case errors.Is(err, ErrVersionMismatch):
		return "version_mismatch"
	case errors.Is(err, ErrTypeMismatch):
		return "type_mismatch"
	case errors.Is(err, ErrInvalidEnvelope):
		return "invalid_envelope"
	default:
		return "unknown"
	}
}
