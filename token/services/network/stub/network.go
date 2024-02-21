/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package stub

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/services/chaincode"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	api2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/rws/keys"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

const (
	InvokeFunction            = "invoke"
	QueryPublicParamsFunction = "queryPublicParams"
	QueryTokensFunctions      = "queryTokens"
	AreTokensSpent            = "areTokensSpent"
)

type lm struct {
	lm *fabric.LocalMembership
}

// TODO: are these the signers of the envelope?

func (n *lm) DefaultIdentity() view.Identity {
	return []byte{}

	//return n.lm.DefaultIdentity()
}

func (n *lm) AnonymousIdentity() view.Identity {
	return []byte{}
	// return n.lm.AnonymousIdentity()
}

type nv struct {
	tokenVault driver.TokenVault
}

func (v *nv) QueryEngine() api2.QueryEngine {
	return v.tokenVault.QueryEngine()
}

func (v *nv) CertificationStorage() api2.CertificationStorage {
	return v.tokenVault.CertificationStorage()
}

func (v *nv) DeleteTokens(ns string, ids ...*token.ID) error {
	return v.tokenVault.DeleteTokens(ns, ids...)
}

func (v *nv) Status(txID string) (driver.ValidationCode, error) {
	// vc, _, err := v.v.Status(txID)
	// return driver.ValidationCode(vc), err
	panic("not implemented")
}

func (v *nv) GetLastTxID() (string, error) {
	panic("not implemented")
}

// UnspentTokensIteratorBy returns an iterator over all unspent tokens by type and id
func (v *nv) UnspentTokensIteratorBy(id, typ string) (network.UnspentTokensIterator, error) {
	return v.tokenVault.QueryEngine().UnspentTokensIteratorBy(id, typ)
}

// UnspentTokensIterator returns an iterator over all unspent tokens
func (v *nv) UnspentTokensIterator() (network.UnspentTokensIterator, error) {
	return v.tokenVault.QueryEngine().UnspentTokensIterator()
}

func (v *nv) ListUnspentTokens() (*token.UnspentTokens, error) {
	return v.tokenVault.QueryEngine().ListUnspentTokens()
}

func (v *nv) Exists(id *token.ID) bool {
	return v.tokenVault.CertificationStorage().Exists(id)
}

func (v *nv) Store(certifications map[*token.ID][]byte) error {
	return v.tokenVault.CertificationStorage().Store(certifications)
}

func (v *nv) DiscardTx(txID string) error {
	// set status to driver.Invalid
	panic("not implemented")
}

type Network struct {
	network     string
	channel     string
	persistence *Persistence
	vault       driver.Vault
	listeners   map[string]driver.TxStatusChangeListener
}

func NewNetwork(network, channel string, tokenVault driver.Vault) *Network {
	return &Network{
		network:     network,
		channel:     channel,
		persistence: new(Persistence),
		vault:       tokenVault,
	}
}

func (n *Network) Name() string {
	return n.network
}

func (n *Network) Channel() string {
	return n.channel
}

func (n *Network) Vault(namespace string) (driver.Vault, error) {
	nv := &nv{
		tokenVault: n.vault,
	}
	return nv, nil
}

type Transaction struct {
	TxID       string
	Status     driver.ValidationCode
	RequestRaw []byte
}
type Persistence struct {
	envelopes    map[string][]byte
	transient    map[string]driver.TransientMap
	transactions map[string]Transaction
}

func (p *Persistence) StoreEnvelope(id string, env []byte) error {
	p.envelopes[id] = env
	return nil
}
func (p *Persistence) EnvelopeExists(id string) bool {
	return p.envelopes[id] != nil
}
func (p *Persistence) StoreTransient(id string, transientmap driver.TransientMap) error {
	p.transient[id] = transientmap
	return nil
}
func (p *Persistence) TransientExists(id string) bool {
	return p.transient[id] != nil
}
func (p *Persistence) GetTransient(id string) (driver.TransientMap, error) {
	return p.transient[id], nil
}
func (p *Persistence) GetPublicParams(namespace string) ([]byte, error) {
	return os.ReadFile("../zkat_dlog.json")
}

// to implement the ledger interface
func (p *Persistence) Status(id string) (driver.ValidationCode, error) {
	tx := p.transactions[id] // TODO if not exists
	return tx.Status, nil
}

type rws struct{}

func (r *rws) Done() {
	logger.Info("done called on RWS")
}
func (n *Network) GetRWSet(id string, results []byte) (driver.RWSet, error) {
	return &rws{}, nil
}

func (n *Network) StoreEnvelope(id string, env []byte) error {
	return n.persistence.StoreEnvelope(id, env)
}

func (n *Network) EnvelopeExists(id string) bool {
	return n.persistence.EnvelopeExists(id)
}

func (n *Network) Broadcast(context context.Context, blob interface{}) error {
	return nil

	// if n.listeners[txID] != nil {

	// }

	// submit endorsed tx to orderer

	// TODO: set status to final, but how do we konw the tx id?

	//return n.n.Ordering().Broadcast(context, blob)
}

func (n *Network) IsFinalForParties(id string, endpoints ...view.Identity) error {
	// TODO this is supposed to wait until it's final.
	//return n.ch.Finality().IsFinalForParties(id, endpoints...)
	return nil
	//tx := n.persistence.transactions[id]
}

func (n *Network) IsFinal(ctx context.Context, id string) error {
	// n.persistence.transactions[id].Status
	return nil
}

func (n *Network) NewEnvelope() driver.Envelope {
	return NewEnvelope()
}

func (n *Network) StoreTransient(id string, transient driver.TransientMap) error {
	return n.persistence.StoreTransient(id, transient)
}

func (n *Network) TransientExists(id string) bool {
	return n.persistence.TransientExists(id)
}

func (n *Network) GetTransient(id string) (driver.TransientMap, error) {
	return n.persistence.GetTransient(id)
}

func (n *Network) RequestApproval(context view.Context, tms *token2.ManagementService, requestRaw []byte, signer view.Identity, txID driver.TxID) (driver.Envelope, error) {
	// transient entry = requestRaw
	// signed by signer
	// endorse returns Envelope (signed by peer?)
	n.persistence.transactions[txID.String()] = Transaction{
		TxID:       txID.String(),
		Status:     driver.Valid, // todo?
		RequestRaw: requestRaw,
	}

	// TODO return envelope
	return nil, nil
}

func (n *Network) ComputeTxID(id *driver.TxID) string {
	logger.Debugf("compute tx id for [%s]", id.String())
	hasher := sha256.New()
	hasher.Write(id.Nonce)
	hasher.Write(id.Creator)
	return hex.EncodeToString(hasher.Sum(nil))
}

func (n *Network) FetchPublicParameters(namespace string) ([]byte, error) {
	return n.persistence.GetPublicParams(namespace)
}

func (n *Network) QueryTokens(context view.Context, namespace string, IDs []*token.ID) ([][]byte, error) {
	// TODO store tokens

	var tokens [][]byte
	for _, id := range IDs {
		s := id.String()
		raw := n.persistence.transactions[s].RequestRaw // TODO QueryTokens != RequestRaw
		tokens = append(tokens, raw)
	}
	return tokens, nil
}

func (n *Network) AreTokensSpent(c view.Context, namespace string, tokenIDs []*token.ID, meta []string) ([]bool, error) {
	// TODO store tokens

	sIDs := make([]string, len(tokenIDs))
	var err error
	for i, id := range tokenIDs {
		sIDs[i], err = keys.CreateTokenKey(id.TxId, id.Index)
		if err != nil {
			return nil, errors.Wrapf(err, "cannot compute spent id for [%v]", id)
		}
	}

	idsRaw, err := json.Marshal(sIDs)
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling ids")
	}

	payloadBoxed, err := c.RunView(chaincode.NewQueryView(
		namespace,
		AreTokensSpent,
		idsRaw,
	).WithNetwork(n.Name()).WithChannel(n.Channel()))
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to query the token chaincode for tokens spent")
	}

	// Unbox
	raw, ok := payloadBoxed.([]byte)
	if !ok {
		return nil, errors.Errorf("expected []byte from TCC, got [%T]", payloadBoxed)
	}
	var spent []bool
	if err := json.Unmarshal(raw, &spent); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal esponse")
	}

	return spent, nil
}

func (n *Network) LocalMembership() driver.LocalMembership {
	return n.LocalMembership()
}

func (n *Network) GetEnrollmentID(raw []byte) (string, error) {
	ai := &idemix.AuditInfo{}
	if err := ai.FromBytes(raw); err != nil {
		return "", errors.Wrapf(err, "failed unamrshalling audit info [%s]", raw)
	}
	return ai.EnrollmentID(), nil
}

func (n *Network) SubscribeTxStatusChanges(txID string, listener driver.TxStatusChangeListener) error {
	n.listeners[txID] = listener

	return nil
	// Remote: /transaction/:id/subscribe
	// should have reconnect option
	// keep track of subscription
}

func (n *Network) UnsubscribeTxStatusChanges(txID string, listener driver.TxStatusChangeListener) error {
	n.listeners[txID] = nil
	return nil
	// Remove subscription / close connection
}

func (n *Network) LookupTransferMetadataKey(namespace string, startingTxID string, key string, timeout time.Duration) ([]byte, error) {
	//transferMetadataKey, err := keys.CreateTransferActionMetadataKey(key)

	// TODO: when adding tx to persistence, we have to store the transfer metadata too (see tcc)
	return nil, nil
}

// Expose the 'Status(id)' function
func (n *Network) Ledger() (driver.Ledger, error) {
	return n.persistence, nil
}

func (n *Network) ProcessNamespace(namespace string) error {
	// Get updates from remote ledger (what's the starting point?)
	return nil
}
