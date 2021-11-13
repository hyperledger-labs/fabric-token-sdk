/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"encoding/base64"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type TransientMap map[string][]byte

type TxID struct {
	Nonce   []byte
	Creator []byte
}

func (t *TxID) String() string {
	return fmt.Sprintf("[%s:%s]", base64.StdEncoding.EncodeToString(t.Nonce), base64.StdEncoding.EncodeToString(t.Creator))
}

type Network interface {
	Name() string
	Channel() string
	Vault(namespace string) (Vault, error)
	GetRWSet(id string, results []byte) (RWSet, error)
	StoreEnvelope(id string, env []byte) error
	Broadcast(blob interface{}) error
	IsFinalForParties(id string, endpoints ...view.Identity) error
	IsFinal(id string) error
	AnonymousIdentity() view.Identity
	NewEnvelope() Envelope
	StoreTransient(id string, transient TransientMap) error
	RequestApproval(context view.Context, namespace string, requestRaw []byte, signer view.Identity, txID TxID) (Envelope, error)
	ComputeTxID(id *TxID) string
	AddIssuer(context view.Context, pk []byte) error
	FetchPublicParameters(namespace string) ([]byte, error)
	RegisterAuditor(context view.Context, namespace string, id view.Identity) error
	RegisterCertifier(context view.Context, namespace string, id view.Identity) error
	QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error)
	LocalMembership() LocalMembership
	GetEnrollmentID(raw []byte) (string, error)
}
