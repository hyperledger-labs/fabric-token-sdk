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
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

var logger = flogging.MustGetLogger("token-sdk.certifier.interactive")

type QueryEngine interface {
	ListUnspentTokens() (*token2.UnspentTokens, error)
}

type CertificationStorage interface {
	Exists(id *token2.ID) bool
	Store(certifications map[*token2.ID][]byte) error
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
	}
}

func (d *CertificationClient) IsCertified(id *token2.ID) bool {
	return d.certificationStorage.Exists(id)
}

func (d *CertificationClient) RequestCertification(ids ...*token2.ID) error {
	var toBeCertified []*token2.ID
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
	certifications, ok := resultBoxed.(map[*token2.ID][]byte)
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
	for {
		// Check the unspent tokens
		tokens, err := d.queryEngine.ListUnspentTokens()
		if err != nil {
			break
		}
		var toBeCertified []*token2.ID
		for _, token := range tokens.Tokens {
			// does token have a certification?
			if !d.certificationStorage.Exists(token.Id) {
				// if no, batch it
				toBeCertified = append(toBeCertified, token.Id)
			}
		}

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

		done := false
		for {
			select {
			case <-d.ctx.Done():
				return
			case <-time.After(2 * time.Second):
				txid, err := d.vault.GetLastTxID()
				if err != nil {
					continue
				}
				if txid != lastTXID {
					lastTXID = txid
					done = true
					break
				}
			}
			if done {
				break
			}
		}
	}
}
