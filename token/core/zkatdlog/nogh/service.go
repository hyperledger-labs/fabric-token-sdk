/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package nogh

import (
	"encoding/base64"
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
	publicParamInitialization sync.RWMutex
	Channel                   Channel
	Namespace                 string
	SP                        view2.ServiceProvider
	pp                        *crypto.PublicParams
	PPLabel                   string
	PublicParamsFetcher       api3.PublicParamsFetcher
	TokenCommitmentLoader     TokenCommitmentLoader
	QE                        QueryEngine
	DeserializerProvider      DeserializerProvider

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

func (s *Service) initOrSetPublicParams() {
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
		logger.Fatalf("Failed fetching %s from namespace %s: %s", setupKey, s.Namespace, err)
	}
	if len(raw) == 0 {
		logger.Warnf("public parameters with key [%s] not found, fetch them", setupKey)
		raw, err = s.PublicParamsFetcher.Fetch()
		if err != nil {
			logger.Fatalf("failed retrieving public params [%s]", err)
		}
	}

	logger.Debugf("unmarshal public parameters with key [%s], len [%d]", setupKey, len(raw))
	s.pp = &crypto.PublicParams{}
	s.pp.Label = s.PPLabel
	err = s.pp.Deserialize(raw)
	if err != nil {
		logger.Fatalf("Failed deserializing public params from %s: %v", base64.StdEncoding.EncodeToString(raw), err)
	}
	logger.Debugf("unmarshal public parameters with key [%s] done", setupKey)

	ip, err := s.pp.GetIssuingPolicy()
	if err != nil {
		logger.Fatalf("Failed obtaining issuing policy: %v", err)
	}
	logger.Debugf("returning public parameters [%d,%d,%d,%d]", len(s.pp.ZKATPedParams), len(ip.Issuers), ip.IssuersNumber, ip.BitLength)
}

func (s *Service) PublicParams() *crypto.PublicParams {
	s.publicParamInitialization.RLock()
	if s.pp == nil {
		s.publicParamInitialization.RUnlock()

		s.publicParamInitialization.Lock()
		defer s.publicParamInitialization.Unlock()

		if s.pp == nil {
			s.initOrSetPublicParams()
		}
		return s.pp
	}

	defer s.publicParamInitialization.RUnlock()

	return s.pp
}

func (s *Service) FetchPublicParams() error {
	s.PublicParams()
	return nil
}


func (s *Service) Deserializer() (api3.Deserializer, error) {
	d, err := s.DeserializerProvider(s.PublicParams())
	if err != nil {
		return nil, err
	}
	return d, nil
}
