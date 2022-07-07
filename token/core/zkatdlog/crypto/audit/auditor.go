/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package audit

import (
	"encoding/asn1"
	"encoding/json"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/exchange"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

var logger = flogging.MustGetLogger("token-sdk.zkatdlog.audit")

// signing identity
type SigningIdentity interface {
	driver.SigningIdentity
}

// deserializer
type Deserializer interface {
	GetOwnerMatcher(raw []byte) (driver.Matcher, error)
}

// AuditableToken contains a zkat token and the information that allows
// an auditor to learn its content.
type AuditableToken struct {
	Token     *token.Token
	data      *tokenDataOpening
	owner     *ownerOpening
	auditInfo []byte
}

func NewAuditableToken(token *token.Token, auditInfo []byte, matcher driver.Matcher, ttype string, value *math.Zr, bf *math.Zr) (*AuditableToken, error) {
	return &AuditableToken{
		Token: token,
		owner: &ownerOpening{
			ownerInfo: matcher,
		},
		data: &tokenDataOpening{
			ttype: ttype,
			value: value,
			bf:    bf,
		},
		auditInfo: auditInfo,
	}, nil
}

// tokenDataOpening contains the opening of the TokenData.
// TokenData is a Pedersen commitment to token type and value.
type tokenDataOpening struct {
	ttype string
	value *math.Zr
	bf    *math.Zr
}

// OwnerOpening contains the information that allows the auditor to identify the owner.
type ownerOpening struct {
	ownerInfo driver.Matcher
}

// Auditor inspects zkat tokens and their owners.
type Auditor struct {
	// Owner Identity Deserializer
	Des Deserializer
	// Auditor's signing identity
	Signer SigningIdentity
	// Pedersen generators used to compute TokenData
	PedersenParams []*math.G1
	// SigningIdentity parameters (e.g., pseudonym parameters)
	NYMParams []byte
	// Elliptic curve
	Curve *math.Curve
}

func NewAuditor(des Deserializer, pp []*math.G1, nymparams []byte, signer SigningIdentity, c *math.Curve) *Auditor {
	return &Auditor{
		Des:            des,
		PedersenParams: pp,
		NYMParams:      nymparams,
		Signer:         signer,
		Curve:          c,
	}
}

// Endorse is called to sign a valid token request
func (a *Auditor) Endorse(tokenRequest *driver.TokenRequest, txID string) ([]byte, error) {
	if tokenRequest == nil {
		return nil, errors.Errorf("audit of tx [%s] failed: : token request is nil", txID)
	}
	// Marshal tokenRequest
	bytes, err := asn1.Marshal(driver.TokenRequest{Issues: tokenRequest.Issues, Transfers: tokenRequest.Transfers})
	if err != nil {
		return nil, errors.Errorf("audit of tx [%s] failed: error marshal token request for signature", txID)
	}
	// Sign
	logger.Debugf("Endorse [%s][%s]", hash.Hashable(bytes).String(), txID)
	if a.Signer == nil {
		return nil, errors.Errorf("audit of tx [%s] failed: signer is nil", txID)
	}

	return a.Signer.Sign(append(bytes, []byte(txID)...))
}

// Check validates TokenRequest against TokenRequestMetadata
func (a *Auditor) Check(tokenRequest *driver.TokenRequest, tokenRequestMetadata *driver.TokenRequestMetadata, inputTokens [][]*token.Token, txID string) error {
	// De-obfuscate issue requests
	outputsFromIssue, err := getAuditInfoForIssues(a.Des, tokenRequest.Issues, tokenRequestMetadata.Issues)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for issues")
	}
	// check validity of issue requests
	err = a.checkIssueRequests(outputsFromIssue, txID)
	if err != nil {
		return errors.Wrapf(err, "failed checking issues")
	}
	// De-odfuscate transfer requests
	auditableInputs, outputsFromTransfer, err := getAuditInfoForTransfers(a.Des, tokenRequest.Transfers, tokenRequestMetadata.Transfers, inputTokens)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for transfers")
	}
	// check validity of transfer requests
	if err := a.checkTransferRequests(auditableInputs, outputsFromTransfer, txID); err != nil {
		return errors.Wrapf(err, "failed checking transfers")
	}

	return nil
}

// checkTransferRequests verifies that the commitments in transfer inputs and outputs match the information provided in the clear.
func (a *Auditor) checkTransferRequests(inputs [][]*AuditableToken, outputsFromTransfer [][]*AuditableToken, txID string) error {

	for k, transferred := range outputsFromTransfer {
		err := a.inspectOutputs(transferred)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th transfer in tx [%s] failed", k, txID)
		}
	}

	for k, i := range inputs {
		err := a.inspectInputs(i)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th transfer in tx [%s] failed", k, txID)
		}
	}

	return nil
}

// checkTransferRequests verifies that the commitments in issue outputs match the information provided in the clear.
func (a *Auditor) checkIssueRequests(outputsFromIssue [][]*AuditableToken, txID string) error {
	// Inspect
	for k, issued := range outputsFromIssue {
		err := a.inspectOutputs(issued)
		if err != nil {
			return errors.Wrapf(err, "audit of %d th issue in tx [%s] failed", k, txID)
		}
	}
	return nil
}

// inspectOutputs verifies that the commitments in an array of outputs match the information provided in the clear.
func (a *Auditor) inspectOutputs(tokens []*AuditableToken) error {
	for i, t := range tokens {
		if t == nil || t.Token == nil {
			return errors.Errorf("failed to inspect nil output [%d]", i)
		}
		err := a.inspectOutput(t, i)
		if err != nil {
			return errors.Wrapf(err, "failed inspecting output [%d]", i)
		}
	}

	return nil
}

// inspectOutput verifies that the commitments in an output token of a given index
// match the information provided in the clear.
func (a *Auditor) inspectOutput(output *AuditableToken, index int) error {
	if len(a.PedersenParams) != 3 {
		return errors.Errorf("length of Pedersen basis != 3")
	}
	if output == nil || output.data == nil {
		return errors.Errorf("invalid output at index [%d]", index)
	}
	t, err := common.ComputePedersenCommitment([]*math.Zr{a.Curve.HashToZr([]byte(output.data.ttype)), output.data.value, output.data.bf}, a.PedersenParams, a.Curve)
	if err != nil {
		return err
	}
	if output.Token == nil {
		return errors.Errorf("output at index [%d] in invalid", index)
	}
	if output.Token == nil || output.Token.Data == nil {
		return errors.Errorf("invalid output at index [%d]", index)
	}
	if !t.Equals(output.Token.Data) {
		return errors.Errorf("output at index [%d] does not match the provided opening", index)
	}
	if !output.Token.IsRedeem() {
		return a.inspectTokenOwner(output, index)
	}
	return nil
}

// inspectInputs verifies that the commitments in an array of inputs match the information provided in the clear.
func (a *Auditor) inspectInputs(inputs []*AuditableToken) error {
	for i, input := range inputs {
		if input == nil || input.Token == nil {
			return errors.Errorf("invalid input at index [%d]", i)
		}

		if !input.Token.IsRedeem() {
			owner, err := a.rawOwner(input.Token.Owner)
			if err != nil {
				return errors.Errorf("input owner at index [%d] cannot be unwrapped", i)
			}
			// this is not a redeem
			if input.owner.ownerInfo == nil {
				return errors.Errorf("invalid input at index [%d]: owner info is nil", i)
			}
			if err := input.owner.ownerInfo.Match(owner); err != nil {
				return errors.Errorf("input at index [%d] does not match the provided opening", i)
			}
		}
			if err := a.inspectTokenOwner(input, i); err != nil {
				return errors.Errorf("invalid input token owner at index [%d]", i)
			}
		}
	return nil
}

type ScriptInfo struct {
	Sender    []byte
	Recipient []byte
}

func (a *Auditor) inspectTokenOwner(token *AuditableToken, index int) error {
	owner, err := identity.UnmarshallRawOwner(token.Token.Owner)
	if err != nil {
		return errors.Errorf("input owner at index [%d] cannot be unmarshalled", index)
	}
	if owner.Type == identity.SerializedIdentityType {
		owner, err := a.rawOwner(token.Token.Owner)
		if err != nil {
			return errors.Errorf("input owner at index [%d] cannot be unwrapped", index)
		}
		if err := token.owner.ownerInfo.Match(owner); err != nil {
			return errors.Errorf("owner at index [%d] does not match the provided opening", index)
		}
		return nil
	}
	if owner.Type != exchange.ScriptTypeExchange {
		return errors.Errorf("invalid owner in token")
	}

	scriptInf := &ScriptInfo{}
	if err := json.Unmarshal(token.auditInfo, scriptInf); err != nil {
		return errors.Wrapf(err, "failed to unmarshal exchange info")
	}
	script := &exchange.Script{}
	err = json.Unmarshal(owner.Identity, script)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal exchange script")
	}

	sender, err := a.Des.GetOwnerMatcher(scriptInf.Sender)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal audit info from script sender [%s]", string(scriptInf.Sender))
	}
	ro, err := identity.UnmarshallRawOwner(script.Sender)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve raw owner from sender in exchange script")
	}
	if err := sender.Match(ro.Identity); err != nil {
		return errors.Wrapf(err, "token at index [%d] does not match the provided opening [%s]", index, string(scriptInf.Sender))
	}

	recipient, err := a.Des.GetOwnerMatcher(scriptInf.Recipient)
	if err != nil {
		return errors.Wrapf(err, "failed to unmarshal audit info from script recipient [%s]", string(scriptInf.Recipient))
	}
	ro, err = identity.UnmarshallRawOwner(script.Recipient)
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve raw owner from recipien in exchange script")
	}
	if err := recipient.Match(ro.Identity); err != nil {
		return errors.Wrapf(err, "token at index [%d] does not match the provided opening [%s]", index, string(scriptInf.Recipient))
	}

	return nil
}

// rawOwner unmarshal a series of bytes into an raw owner
func (a *Auditor) rawOwner(raw []byte) ([]byte, error) {
	si, err := identity.UnmarshallRawOwner(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to identity.RawOwner{}")
	}
	return si.Identity, nil
}

// getAuditInfoForIssues returns an array of AuditableToken for each issue action
// It takes a deserializer, an array of serialized issue actions and an array of issue metadata.
func getAuditInfoForIssues(des Deserializer, issues [][]byte, metadata []driver.IssueMetadata) ([][]*AuditableToken, error) {
	if len(issues) != len(metadata) {
		return nil, errors.Errorf("number of issues does not match number of provided metadata")
	}
	outputs := make([][]*AuditableToken, len(issues))
	for k, md := range metadata {
		ia := &issue2.IssueAction{}
		err := json.Unmarshal(issues[k], ia)
		if err != nil {
			return nil, err
		}
		if &md == nil {
			return nil, errors.Errorf("invalid issue metadata: it is nil")
		}
		if len(ia.OutputTokens) != len(md.ReceiversAuditInfos) || len(ia.OutputTokens) != len(md.TokenInfo) {
			return nil, errors.Errorf("number of output does not match number of provided metadata")
		}
		for i := 0; i < len(md.ReceiversAuditInfos); i++ {
			ti := &token.TokenInformation{}
			err := json.Unmarshal(md.TokenInfo[i], ti)
			if err != nil {
				return nil, err
			}
			matcher, err := des.GetOwnerMatcher(md.ReceiversAuditInfos[i])
			if err != nil {
				return nil, err
			}
			if ia.OutputTokens[i] == nil {
				return nil, errors.Errorf("output token at index [%d] is nil", i)
			}
			if ia.OutputTokens[i].IsRedeem() {
				return nil, errors.Errorf("issue cannot redeem tokens")
			}
			ao, err := NewAuditableToken(ia.OutputTokens[i], md.ReceiversAuditInfos[i], matcher, ti.Type, ti.Value, ti.BlindingFactor)
			if err != nil {
				return nil, err
			}
			outputs[k] = append(outputs[k], ao)
		}
	}
	return outputs, nil
}

// getAuditInfoForTransfers returns an array of AuditableToken for each transfer action.
// It takes a deserializer, an array of serialized transfer actions and an array of transfer metadata.
func getAuditInfoForTransfers(des Deserializer, transfers [][]byte, metadata []driver.TransferMetadata, inputs [][]*token.Token) ([][]*AuditableToken, [][]*AuditableToken, error) {
	if len(transfers) != len(metadata) {
		return nil, nil, errors.Errorf("number of transfers does not match the number of provided metadata")
	}
	if len(inputs) != len(metadata) {
		return nil, nil, errors.Errorf("number of inputs does not match the number of provided metadata")
	}
	auditableInputs := make([][]*AuditableToken, len(inputs))
	outputs := make([][]*AuditableToken, len(transfers))
	for k, tr := range metadata {
		if len(tr.SenderAuditInfos) != len(inputs[k]) {
			return nil, nil, errors.Errorf("number of inputs does not match the number of senders")
		}
		for i := 0; i < len(tr.SenderAuditInfos); i++ {
			var matcher driver.Matcher
			var err error
			if inputs[k][i] == nil {
				return nil, nil, errors.Errorf("input[%d][%d] is nil", k, i)
			}
			if !inputs[k][i].IsRedeem() {
				matcher, err = des.GetOwnerMatcher(tr.SenderAuditInfos[i])
				if err != nil {
					return nil, nil, err
				}
			}
			ai, err := NewAuditableToken(inputs[k][i], tr.SenderAuditInfos[i], matcher, "", nil, nil)
			if err != nil {
				return nil, nil, err
			}
			auditableInputs[k] = append(auditableInputs[k], ai)
		}
		ta := &transfer.TransferAction{}
		err := json.Unmarshal(transfers[k], ta)
		if err != nil {
			return nil, nil, err
		}
		if len(ta.OutputTokens) != len(tr.ReceiverAuditInfos) {
			return nil, nil, errors.Errorf("number of outputs does not match the number of receivers")
		}
		for i := 0; i < len(tr.ReceiverAuditInfos); i++ {
			ti := &token.TokenInformation{}
			err := json.Unmarshal(tr.TokenInfo[i], ti)
			if err != nil {
				return nil, nil, err
			}

			var matcher driver.Matcher
			if ta.OutputTokens[i] == nil {
				return nil, nil, errors.Errorf("output token at index [%d] is nil", i)
			}
			if !ta.OutputTokens[i].IsRedeem() {
				matcher, err = des.GetOwnerMatcher(tr.ReceiverAuditInfos[i])
				if err != nil {
					return nil, nil, err
				}
			}
			ao, err := NewAuditableToken(ta.OutputTokens[i], tr.ReceiverAuditInfos[i], matcher, ti.Type, ti.Value, ti.BlindingFactor)
			if err != nil {
				return nil, nil, err
			}
			outputs[k] = append(outputs[k], ao)
		}
	}
	return auditableInputs, outputs, nil
}
