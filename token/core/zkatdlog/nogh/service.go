/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"sync"

	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	api3 "github.com/hyperledger-labs/fabric-token-sdk/token/api"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	info = "info"
)

type Channel interface {
	Name() string
	Vault() *fabric.Vault
}

type TokenCommitmentLoader interface {
	GetTokenCommitments(ids []*token3.Id) ([]*token.Token, error)
}

type QueryEngine interface {
	IsMine(id *token3.Id) (bool, error)
	ListUnspentTokens() (*token3.UnspentTokens, error)
	ListAuditTokens(ids ...*token3.Id) ([]*token3.Token, error)
	ListHistoryIssuedTokens() (*token3.IssuedTokens, error)
}

type service struct {
	channel               Channel
	namespace             string
	sp                    view2.ServiceProvider
	pp                    *crypto.PublicParams
	publicParamsFetcher   api3.PublicParamsFetcher
	tokenCommitmentLoader TokenCommitmentLoader
	qe                    QueryEngine

	issuers []*struct {
		label string
		index int
		sk    *bn256.Zr
		pk    *bn256.G1
		fID   view.Identity
	}

	identityProvider api3.IdentityProvider
	ownerWallets     []*wallet
	issuerWallets    []*issuerWallet
	auditorWallets   []*auditorWallet
	walletsLock      sync.Mutex
}

func NewTokenService(
	channel Channel,
	namespace string,
	sp view2.ServiceProvider,
	publicParamsFetcher api3.PublicParamsFetcher,
	tokenCommitmentLoader TokenCommitmentLoader,
	queryEngine QueryEngine,
	identityProvider api3.IdentityProvider,
) (*service, error) {
	s := &service{
		channel:               channel,
		namespace:             namespace,
		sp:                    sp,
		publicParamsFetcher:   publicParamsFetcher,
		tokenCommitmentLoader: tokenCommitmentLoader,
		qe:                    queryEngine,
		identityProvider:      identityProvider,
	}
	return s, nil
}

func (s *service) DeserializeToken(tok []byte, infoRaw []byte) (*token3.Token, view.Identity, error) {
	output := &token.Token{}
	if err := output.Deserialize(tok); err != nil {
		return nil, nil, err
	}

	ti := &token.TokenInformation{}
	err := ti.Deserialize(infoRaw)
	if err != nil {
		return nil, nil, err
	}

	to, err := output.GetTokenInTheClear(ti, s.PublicParams())
	if err != nil {
		return nil, nil, err
	}

	return to, ti.Issuer, nil
}

func (s *service) IdentityProvider() api3.IdentityProvider {
	return s.identityProvider
}

func (s *service) Validator() api3.Validator {
	return validator.New(s.PublicParams())
}

func (s *service) PublicParamsManager() api3.PublicParamsManager {
	return ppm.New(s.PublicParams())
}

func (s *service) PublicParams() *crypto.PublicParams {
	if s.pp == nil {
		// load
		qe, err := s.channel.Vault().NewQueryExecutor()
		if err != nil {
			panic(err)
		}
		defer qe.Done()

		setupKey, err := keys.CreateSetupKey()
		if err != nil {
			panic(err)
		}
		logger.Debugf("get public parameters with key [%s]", setupKey)
		raw, err := qe.GetState(s.namespace, setupKey)
		if err != nil {
			panic(err)
		}
		if len(raw) == 0 {
			logger.Warnf("public parameters with key [%s] not found, fetch them", setupKey)
			raw, err = s.publicParamsFetcher.Fetch()
			if err != nil {
				logger.Errorf("failed retrieving public params [%s]", err)
				return nil
			}
		}

		logger.Debugf("unmarshal public parameters with key [%s], len [%d]", setupKey, len(raw))
		s.pp = &crypto.PublicParams{}
		err = s.pp.Deserialize(raw)
		if err != nil {
			panic(err)
		}
		logger.Debugf("unmarshal public parameters with key [%s] done", setupKey)
	}

	ip, err := s.pp.GetIssuingPolicy()
	if err != nil {
		panic(err)
	}
	logger.Debugf("returning public parameters [%d,%d,%d,%d]", len(s.pp.ZKATPedParams), len(ip.Issuers), ip.IssuersNumber, ip.BitLength)

	return s.pp
}

func (s *service) FetchPublicParams() error {
	raw, err := s.publicParamsFetcher.Fetch()
	if err != nil {
		return errors.WithMessagef(err, "failed fetching public params from fabric")
	}

	pp := &crypto.PublicParams{}
	err = pp.Deserialize(raw)
	if err != nil {
		return errors.Wrapf(err, "failed deserializing public params")
	}

	ip, err := pp.GetIssuingPolicy()
	if err != nil {
		return errors.Wrapf(err, "failed deserializing issuing policy")
	}
	logger.Debugf("fetching public parameters done, issue policy [%d,%d,%d]", len(ip.Issuers), ip.IssuersNumber, ip.BitLength)

	s.pp = pp
	return nil
}
