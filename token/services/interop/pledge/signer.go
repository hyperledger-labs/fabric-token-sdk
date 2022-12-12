/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"encoding/json"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

// Signer for a pledge script
type Signer struct {
	// Sender corresponds to the sender of the token at time of pledge
	// this is used during reclaim
	Sender driver.Signer
	// Issuer corresponds to the issuer in the origin network
	// this is used during redeem
	Issuer driver.Signer
	// IssuerSignature is the signature from the issuer in the origin network
	// it attests to whether a pledged token has been successfully claimed or not
	// this is used during reclaim
	IssuerSignature []byte
}

// Signature encodes the signature that spends a pledge script
type Signature struct {
	Reclaim bool
	// this is empty in case of redeem
	SenderSignature []byte
	IssuerSignature []byte
}

func (s *Signer) Sign(message []byte) ([]byte, error) {
	sigma := Signature{}
	var err error
	if s.Issuer == nil {
		if s.Sender == nil {
			return nil, errors.New("please initialize pledge signer correctly: empty sender")
		}
		sigma.Reclaim = true
		sigma.IssuerSignature = s.IssuerSignature

		message = append(message, s.IssuerSignature...)
		sigma.SenderSignature, err = s.Sender.Sign(message)
		if err != nil {
			return nil, err
		}
		logger.Debugf("reclaim signature on message [%s]", hash.Hashable(message).String())
	} else {
		sigma.Reclaim = false
		sigma.IssuerSignature, err = s.Issuer.Sign(message)
		if err != nil {
			return nil, err
		}
		logger.Debugf("redeem signature on message [%s]", hash.Hashable(message).String())
	}
	raw, err := json.Marshal(sigma)
	if err != nil {
		return nil, err
	}
	return raw, nil
}

// Verifier of a Signature it is uniquely linked to a the pledge script identified by
// PledgeID
type Verifier struct {
	// Sender in the pledge script
	Sender driver.Verifier
	// Issuer is the issuer in the pledge script
	Issuer driver.Verifier
	// PledgeID identifies the pledge script
	PledgeID string
}

// Verify checks if a signature is a valid signature on the message with respect to TransferVerifier
func (v *Verifier) Verify(message, sigma []byte) error {
	sig := &Signature{}
	err := json.Unmarshal(sigma, sig)
	if err != nil {
		return errors.Wrapf(err, "failed unmarshalling signature [%s] on message [%s]", hash.Hashable(sigma).String(), hash.Hashable(message).String())
	}
	// this is a redeem
	if !sig.Reclaim {
		if err := v.Issuer.Verify(message, sig.IssuerSignature); err != nil {
			return errors.Wrapf(err, "failed verifying signature [%s] on message [%s], this is not a valid redeem", hash.Hashable(sigma).String(), hash.Hashable(message).String())
		}
		return nil
	}
	// this is a reclaim
	message = append(message, sig.IssuerSignature...)
	if err := v.Sender.Verify(message, sig.SenderSignature); err != nil {
		return errors.Wrapf(err, "failed verifying signature [%s] on message [%s], this is not a valid reclaim", hash.Hashable(sigma).String(), hash.Hashable(message).String())
	}
	err = v.Issuer.Verify([]byte(v.PledgeID), sig.IssuerSignature)
	if err != nil {
		return errors.Wrapf(err, "failed verifying reclaim issuer signature [%s:%s]", v.PledgeID, hash.Hashable(sig.IssuerSignature).String())
	}
	return nil
}
