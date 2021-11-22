/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package approver

import (
	"crypto/rand"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
)

var logger = flogging.MustGetLogger("token.tms.zkat.approver")

type Vault interface {
	NewQueryExecutor() (driver.Executor, error)
	NewRWSet(txid string) (driver.RWSet, error)
}

type Verifier interface {
	Verify(message, sigma []byte) error
}

type SignatureProvider = func(id view.Identity, verifier Verifier) error

type approver struct {
	vault     Vault
	validator translator.Validator
	TxID      string
	rwset     translator.RWSet
	namespace string
}

func NewTokenRWSetApprover(validator translator.Validator, vault Vault, txID string, RWSet driver.RWSet, namespace string) *approver {
	return &approver{
		vault:     vault,
		TxID:      txID,
		rwset:     RWSet,
		validator: validator,
		namespace: namespace,
	}
}

func (v *approver) Validate(sp SignatureProvider, tokenRequest *token.Request) error {
	logger.Debugf("approve token request for tx [%d]", v.TxID)

	logger.Debugf("verify token request for tx [%d]", v.TxID)
	// verify token request
	qe, err := v.vault.NewQueryExecutor()
	if err != nil {
		return errors.Wrap(err, "failed getting query executor")
	}
	defer qe.Done()
	backend := &backend{qe: qe, sp: sp, namespace: v.namespace}
	actions, err := v.validator.Verify(backend, backend, v.TxID, tokenRequest)
	if err != nil {
		return errors.Wrap(err, "failed verifying token request")
	}
	qe.Done()

	logger.Debugf("verify rws for tx [%d]", v.TxID)
	rwset, err := v.vault.NewRWSet(getRandomId())
	if err != nil {
		return errors.Wrap(err, "failed creating new rws")
	}
	// TODO: use a proper issuerValidator
	issuingValidator := &allIssuersValid{}
	translator := translator.New(issuingValidator, v.TxID, rwset, v.namespace)
	for _, action := range actions {
		err = translator.Write(action)
		if err != nil {
			return errors.Wrap(err, "failed writing token action")
		}
	}
	tokenRequestRaw, err := tokenRequest.RequestToBytes()
	if err != nil {
		return errors.Wrap(err, "failed serializing token request")
	}
	err = translator.CommitTokenRequest(tokenRequestRaw, false)
	if err != nil {
		return errors.Wrap(err, "failed writing token request")
	}

	if err := rwset.Equals(v.rwset, v.namespace); err != nil {
		return errors.Wrap(err, "invalid rws, regenerate rws does not match")
	}

	logger.Debugf("approve token request for tx [%d] done", v.TxID)
	return nil
}

type allIssuersValid struct{}

func (i *allIssuersValid) Validate(creator view.Identity, tokenType string) error {
	return nil
}

type backend struct {
	qe        driver.Executor
	sp        SignatureProvider
	namespace string
}

func (b *backend) HasBeenSignedBy(id view.Identity, verifier token.Verifier) error {
	return b.sp(id, verifier)
}

func (b *backend) GetState(key string) ([]byte, error) {
	v, err := b.qe.GetState(b.namespace, key)
	logger.Debugf("Got State [%s,%s] -> [%v]", b.namespace, key, hash.Hashable(v).String())
	return v, err
}

func getRandomId() string {
	key := make([]byte, 24)

	_, err := rand.Read(key)
	if err != nil {
		panic(err)
	}
	return string(key)
}
