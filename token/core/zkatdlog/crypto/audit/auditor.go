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
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/transfer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

// signing identity
type SigningIdentity interface {
	driver.SigningIdentity
}

var logger = flogging.MustGetLogger("token-sdk.zkatdlog.audit")

type AuditableToken struct {
	Token *token.Token
	data  *tokenDataOpening
	owner *ownerOpening
}

func NewAuditableToken(token *token.Token, matcher driver.Matcher, ttype string, value *math.Zr, bf *math.Zr) (*AuditableToken, error) {
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
	}, nil
}

type tokenDataOpening struct {
	ttype string
	value *math.Zr
	bf    *math.Zr
}

type Deserializer interface {
	GetOwnerMatcher(raw []byte) (driver.Matcher, error)
}

type ownerOpening struct {
	ownerInfo driver.Matcher
}

type Auditor struct {
	Des            Deserializer
	Signer         SigningIdentity
	PedersenParams []*math.G1
	NYMParams      []byte
	Curve          *math.Curve
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

func (a *Auditor) Endorse(tokenRequest *driver.TokenRequest, txID string) ([]byte, error) {
	// Prepare signature
	bytes, err := asn1.Marshal(driver.TokenRequest{Issues: tokenRequest.Issues, Transfers: tokenRequest.Transfers})
	if err != nil {
		return nil, errors.Errorf("audit of tx [%s] failed: error marshal token request for signature", txID)
	}
	logger.Debugf("Endorse [%s][%s]", hash.Hashable(bytes).String(), txID)
	return a.Signer.Sign(append(bytes, []byte(txID)...))

}

func (a *Auditor) Check(tokenRequest *driver.TokenRequest, tokenRequestMetadata *driver.TokenRequestMetadata, inputTokens [][]*token.Token, txID string) error {
	outputsFromIssue, err := getAuditInfoForIssues(a.Des, tokenRequest.Issues, tokenRequestMetadata.Issues)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for issues")
	}
	err = a.checkIssueRequests(outputsFromIssue, txID)
	if err != nil {
		return errors.Wrapf(err, "failed checking issues")
	}

	auditableInputs, outputsFromTransfer, err := getAuditInfoForTransfers(a.Des, tokenRequest.Transfers, tokenRequestMetadata.Transfers, inputTokens)
	if err != nil {
		return errors.Wrapf(err, "failed getting audit info for transfers")
	}
	if err := a.checkTransferRequests(auditableInputs, outputsFromTransfer, txID); err != nil {
		return errors.Wrapf(err, "failed checking transfers")
	}

	return nil
}

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

func (a *Auditor) inspectOutputs(tokens []*AuditableToken) error {
	for i, t := range tokens {
		err := a.inspectOutput(t, i)
		if err != nil {
			return errors.Wrapf(err, "failed inspecting output [%d]", i)
		}
		if !t.Token.IsRedeem() { // this is not a redeemed output
			owner, err := a.rawOwner(t.Token.Owner)
			if err != nil {
				return errors.Errorf("output owner at index [%d] cannot be unwrapped", i)
			}
			err = t.owner.ownerInfo.Match(owner)
			if err != nil {
				return errors.Wrapf(err, "output at index [%d] does not match the provided opening", i)
			}
		}
	}

	return nil
}

func (a *Auditor) inspectOutput(output *AuditableToken, index int) error {
	if len(a.PedersenParams) != 3 {
		return errors.Errorf("length of Pedersen basis != 3")
	}
	t, err := common.ComputePedersenCommitment([]*math.Zr{a.Curve.HashToZr([]byte(output.data.ttype)), output.data.value, output.data.bf}, a.PedersenParams, a.Curve)
	if err != nil {
		return err
	}
	if !t.Equals(output.Token.Data) {
		return errors.Errorf("output at index [%d] does not match the provided opening", index)
	}
	return nil
}

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
			if err := input.owner.ownerInfo.Match(owner); err != nil {
				return errors.Errorf("input at index [%d] does not match the provided opening", i)
			}
		}
	}
	return nil
}

func (a *Auditor) rawOwner(raw []byte) ([]byte, error) {
	si, err := identity.UnmarshallRawOwner(raw)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal to identity.RawOwner{}")
	}
	return si.Identity, nil
}

func getAuditInfoForIssues(des Deserializer, issues [][]byte, metadata []driver.IssueMetadata) ([][]*AuditableToken, error) {
	if len(issues) != len(metadata) {
		return nil, errors.Errorf("number of issues does not match number of provided metadata")
	}
	outputs := make([][]*AuditableToken, len(issues))
	for k, issue := range metadata {
		ia := &issue2.IssueAction{}
		err := json.Unmarshal(issues[k], ia)
		if err != nil {
			return nil, err
		}
		if len(ia.OutputTokens) != len(issue.AuditInfos) || len(ia.OutputTokens) != len(issue.TokenInfo) {
			return nil, errors.Errorf("number of output does not match number of provided metadata")
		}
		for i := 0; i < len(issue.AuditInfos); i++ {
			ti := &token.TokenInformation{}
			err := json.Unmarshal(issue.TokenInfo[i], ti)
			if err != nil {
				return nil, err
			}
			matcher, err := des.GetOwnerMatcher(issue.AuditInfos[i])
			if err != nil {
				return nil, err
			}
			if ia.OutputTokens[i].IsRedeem() {
				return nil, errors.Errorf("issue cannot redeem tokens")
			}
			ao, err := NewAuditableToken(ia.OutputTokens[i], matcher, ti.Type, ti.Value, ti.BlindingFactor)
			if err != nil {
				return nil, err
			}
			outputs[k] = append(outputs[k], ao)
		}
	}
	return outputs, nil
}

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
			if !inputs[k][i].IsRedeem() {
				matcher, err = des.GetOwnerMatcher(tr.SenderAuditInfos[i])
				if err != nil {
					return nil, nil, err
				}
			}
			ai, err := NewAuditableToken(inputs[k][i], matcher, "", nil, nil)
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
			if !ta.OutputTokens[i].IsRedeem() {
				matcher, err = des.GetOwnerMatcher(tr.ReceiverAuditInfos[i])
				if err != nil {
					return nil, nil, err
				}
			}
			ao, err := NewAuditableToken(ta.OutputTokens[i], matcher, ti.Type, ti.Value, ti.BlindingFactor)
			if err != nil {
				return nil, nil, err
			}
			outputs[k] = append(outputs[k], ao)
		}
	}
	return auditableInputs, outputs, nil
}
