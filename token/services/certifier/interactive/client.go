/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interactive

import (
	"context"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("token-sdk.certifier.interactive")

type QueryEngine interface {
	UnspentTokensIterator() (network.UnspentTokensIterator, error)
}

type CertificationStorage interface {
	Exists(id *token.ID) bool
	Store(certifications map[*token.ID][]byte) error
}

type Vault interface {
	GetLastTxID() (string, error)
}

type ViewManager interface {
	InitiateView(view view.View) (interface{}, error)
}

type ConfigService interface {
	GetStringSlice(key string) []string
}

type Resolver interface {
	ResolveIdentities(endpoints ...string) []view.Identity
}

// CertificationClient scans the vault for tokens not yet certified and asks the certification.
type CertificationClient struct {
	ctx                  context.Context
	channel, namespace   string
	vault                Vault
	queryEngine          QueryEngine
	certificationStorage CertificationStorage
	viewManager          ViewManager
	certifiers           []view2.Identity
	// waitTime is used in case of a failure. It tells how much time to wait before retrying.
	waitTime time.Duration
}

func NewCertificationClient(
	ctx context.Context,
	channel string,
	namespace string,
	v Vault,
	qe QueryEngine,
	cm CertificationStorage,
	fm ViewManager,
	certifiers []view2.Identity,
) *CertificationClient {
	return &CertificationClient{
		ctx:                  ctx,
		channel:              channel,
		namespace:            namespace,
		vault:                v,
		queryEngine:          qe,
		certificationStorage: cm,
		viewManager:          fm,
		certifiers:           certifiers,
		waitTime:             10 * time.Second,
	}
}

func (d *CertificationClient) IsCertified(id *token.ID) bool {
	return d.certificationStorage.Exists(id)
}

func (d *CertificationClient) RequestCertification(ids ...*token.ID) error {
	var toBeCertified []*token.ID
	for _, id := range ids {
		if !d.IsCertified(id) {
			toBeCertified = append(toBeCertified, id)
		}
	}
	if len(toBeCertified) == 0 {
		// all tokens already certified.
		return nil
	}

	resultBoxed, err := d.viewManager.InitiateView(NewCertificationRequestView(d.channel, d.namespace, d.certifiers[0], toBeCertified...))
	if err != nil {
		return err
	}
	certifications, ok := resultBoxed.(map[*token.ID][]byte)
	if !ok {
		panic("invalid type, expected map[token.ID][]byte")
	}
	if err := d.certificationStorage.Store(certifications); err != nil {
		return err
	}
	return nil
}

func (d *CertificationClient) Start() error {
	go d.Scan()
	return nil
}

func (d *CertificationClient) Scan() {
	var lastTXID string
	var tokens driver.UnspentTokensIterator
	for {
		if tokens != nil {
			tokens.Close()
		}

		logger.Debugf("check the certification of unspent tokens...")
		// Check the unspent tokens
		var err error
		tokens, err = d.queryEngine.UnspentTokensIterator()
		if err != nil {
			logger.Errorf("failed to get an iterator over unspent tokens, wait and retry [%s]", err)
			time.Sleep(d.waitTime)
			continue
		}

		var toBeCertified []*token.ID
		for {
			token, err := tokens.Next()
			if err != nil {
				logger.Errorf("failed to get next unspent tokens, stop here [%s]", err)
				break
			}
			if token == nil {
				break
			}

			// does token have a certification?
			if !d.certificationStorage.Exists(token.Id) {
				// if no, batch it
				toBeCertified = append(toBeCertified, token.Id)
			}
		}
		tokens.Close()

		if len(toBeCertified) != 0 {
			// Request certification
			logger.Debugf("request certification of [%v]", toBeCertified)
			if err := d.RequestCertification(toBeCertified...); err != nil {
				logger.Errorf("failed retrieving certification [%s], try later", err)
				time.Sleep(2 * time.Second)
				continue
			}
			logger.Debugf("request certification of [%v] satisfied with no error", toBeCertified)
		}

		// wait for new tokens to appear in the ledger
		nextTxId, ok := d.getNextCommittedTxID(lastTXID)
		if !ok {
			return
		}
		lastTXID = nextTxId
	}
}

func (d *CertificationClient) getNextCommittedTxID(lastTXID string) (string, bool) {
	timeout := time.NewTimer(2 * time.Second)
	defer timeout.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return "", false
		case <-timeout.C:
			txid, err := d.vault.GetLastTxID()
			if err == nil && txid != lastTXID {
				return txid, true
			}
		}
	}
}
