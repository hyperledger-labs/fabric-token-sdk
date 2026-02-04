/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/fabricutils"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/transaction"
	cb "github.com/hyperledger/fabric-protos-go-apiv2/common"
	"github.com/hyperledger/fabric-x-committer/api/protoblocktx"
	"github.com/hyperledger/fabric-x-committer/utils/signature"
	"github.com/hyperledger/fabric-x-common/protoutil"
)

type Submitter interface {
	Submit(network, channel string, tx *protoblocktx.Tx) error
}

const (
	finalityEOFRetries    = 5
	finalityRetryDuration = 2 * time.Second
)

func NewSubmitterFromFNS(fnsp *fabric.NetworkServiceProvider) *submitter {
	return NewSubmitter(&fnsSigningIdentityProvider{fnsProvider: fnsp}, &fnsBroadcaster{fnsProvider: fnsp})
}

func NewSubmitter(signingIdentityProvider SigningIdentityProvider, envelopeBroadcaster EnvelopeBroadcaster) *submitter {
	return NewSubmitterCustomTxID(signingIdentityProvider, envelopeBroadcaster, protoutil.ComputeTxID)
}

func NewSubmitterCustomTxID(signingIdentityProvider SigningIdentityProvider, envelopeBroadcaster EnvelopeBroadcaster, txIDCalculator func(nonce, creator []byte) string) *submitter {
	return &submitter{
		txIDCalculator:          txIDCalculator,
		signingIdentityProvider: signingIdentityProvider,
		envelopeBroadcaster:     envelopeBroadcaster,
	}
}

type submitter struct {
	txIDCalculator          func(nonce, creator []byte) string
	signingIdentityProvider SigningIdentityProvider
	envelopeBroadcaster     EnvelopeBroadcaster
}

func (s *submitter) Submit(network, channel string, tx *protoblocktx.Tx) error {
	logger.Infof("Submitting to [%s,%s] following %d namespaces: [%v]", network, channel, len(tx.GetNamespaces()), tx.GetNamespaces())

	signer, err := s.signingIdentityProvider.DefaultSigningIdentity(network, channel)
	if err != nil {
		return err
	}

	serializedCreator, err := s.signingIdentityProvider.DefaultIdentity(network, channel)
	if err != nil {
		return err
	}

	nonce, err := transaction.GetRandomNonce()
	if err != nil {
		return errors.Wrapf(err, "failed getting random nonce")
	}

	txID := s.txIDCalculator(nonce, serializedCreator)

	tx.Signatures = make([][]byte, len(tx.GetNamespaces()))
	for idx, ns := range tx.GetNamespaces() {
		// Note that a default msp signer hash the msg before signing.
		// For that reason we use the TxNamespace message as ASN1 encoded msg
		digest, err := signature.ASN1MarshalTxNamespace(txID, ns)
		if err != nil {
			return errors.Wrap(err, "failed asn1 marshal tx")
		}

		sig, err := signer.Sign(digest)
		if err != nil {
			return errors.Wrap(err, "failed signing tx")
		}
		tx.Signatures[idx] = sig
	}

	txRaw, err := proto.Marshal(tx)
	if err != nil {
		return errors.Wrapf(err, "failed marshaling transaction")
	}

	signatureHeader := &cb.SignatureHeader{Creator: serializedCreator, Nonce: nonce}
	channelHeader := protoutil.MakeChannelHeader(cb.HeaderType_MESSAGE, 0, channel, 0)
	channelHeader.TxId = txID
	payloadHeader := protoutil.MakePayloadHeader(channelHeader, signatureHeader)
	env, err := fabricutils.CreateEnvelope(signer, payloadHeader, txRaw)
	if err != nil {
		return errors.Wrapf(err, "failed creating envelope")
	}

	return s.envelopeBroadcaster.Broadcast(network, channel, txID, env)
}
