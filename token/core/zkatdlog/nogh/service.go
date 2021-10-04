/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"sync"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/ppm"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/validator"
	api3 "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/keys"
	token3 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
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

type DeserializerProvider = func(params *crypto.PublicParams) (api3.Deserializer, error)

type Service struct {
	Channel               Channel
	Namespace             string
	SP                    view2.ServiceProvider
	PP                    *crypto.PublicParams
	PPLabel               string
	PublicParamsFetcher   api3.PublicParamsFetcher
	TokenCommitmentLoader TokenCommitmentLoader
	QE                    QueryEngine
	DeserializerProvider  DeserializerProvider

	Issuers []*struct {
		label string
		index int
		sk    *math.Zr
		pk    *math.G1
		fID   view.Identity
	}

	identityProvider api3.IdentityProvider
	OwnerWallets     []*wallet
	IssuerWallets    []*issuerWallet
	AuditorWallets   []*auditorWallet
	WalletsLock      sync.Mutex
}

func NewTokenService(
	channel Channel,
	namespace string,
	sp view2.ServiceProvider,
	publicParamsFetcher api3.PublicParamsFetcher,
	tokenCommitmentLoader TokenCommitmentLoader,
	queryEngine QueryEngine,
	identityProvider api3.IdentityProvider,
	deserializerProvider DeserializerProvider,
	ppLabel string,
) (*Service, error) {
	s := &Service{
		Channel:               channel,
		Namespace:             namespace,
		SP:                    sp,
		PublicParamsFetcher:   publicParamsFetcher,
		TokenCommitmentLoader: tokenCommitmentLoader,
		QE:                    queryEngine,
		identityProvider:      identityProvider,
		DeserializerProvider:  deserializerProvider,
		PPLabel:               ppLabel,
	}
	return s, nil
}

func (s *Service) DeserializeToken(tok []byte, infoRaw []byte) (*token3.Token, view.Identity, error) {
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

func (s *Service) IdentityProvider() api3.IdentityProvider {
	return s.identityProvider
}

func (s *Service) Validator() api3.Validator {
	d, err := s.Deserializer()
	if err != nil {
		panic(err)
	}
	return validator.New(
		s.PublicParams(),
		d,
	)
}

func (s *Service) PublicParamsManager() api3.PublicParamsManager {
	return ppm.New(s.PublicParams())
}

func (s *Service) PublicParams() *crypto.PublicParams {
	if s.PP == nil {
		// load
		qe, err := s.Channel.Vault().NewQueryExecutor()
		if err != nil {
			panic(err)
		}
		defer qe.Done()

		setupKey, err := keys.CreateSetupKey()
		if err != nil {
			panic(err)
		}
		logger.Debugf("get public parameters with key [%s]", setupKey)
		raw, err := qe.GetState(s.Namespace, setupKey)
		if err != nil {
			panic(err)
		}
		if len(raw) == 0 {
			logger.Warnf("public parameters with key [%s] not found, fetch them", setupKey)
			raw, err = s.PublicParamsFetcher.Fetch()
			if err != nil {
				logger.Errorf("failed retrieving public params [%s]", err)
				return nil
			}
		}

		logger.Debugf("unmarshal public parameters with key [%s], len [%d]", setupKey, len(raw))
		s.PP = &crypto.PublicParams{}
		s.PP.Label = s.PPLabel
		err = s.PP.Deserialize(raw)
		if err != nil {
			panic(err)
		}
		logger.Debugf("unmarshal public parameters with key [%s] done", setupKey)
	}

	ip, err := s.PP.GetIssuingPolicy()
	if err != nil {
		panic(err)
	}
	logger.Debugf("returning public parameters [%d,%d,%d,%d]", len(s.PP.ZKATPedParams), len(ip.Issuers), ip.IssuersNumber, ip.BitLength)

	return s.PP
}

func (s *Service) FetchPublicParams() error {
	raw, err := s.PublicParamsFetcher.Fetch()
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

	s.PP = pp
	return nil
}

func (s *Service) Deserializer() (api3.Deserializer, error) {
	d, err := s.DeserializerProvider(s.PublicParams())
	if err != nil {
		return nil, err
	}
	return d, nil
}
