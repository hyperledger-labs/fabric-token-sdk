/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tcc

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"runtime/debug"
	"sync"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracing"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/tracker/metrics"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/translator"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/hyperledger/fabric-chaincode-go/shim"
	pb "github.com/hyperledger/fabric-protos-go/peer"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.tcc")

const (
	InvokeFunction            = "invoke"
	QueryPublicParamsFunction = "queryPublicParams"
	AddCertifierFunction      = "addCertifier"
	QueryTokensFunctions      = "queryTokens"

	PublicParamsPathVarEnv = "PUBLIC_PARAMS_FILE_PATH"
)

type Agent interface {
	EmitKey(val float32, event ...string)
}

type SetupAction struct {
	SetupParameters []byte
}

func (a *SetupAction) GetSetupParameters() ([]byte, error) {
	return a.SetupParameters, nil
}

type AllIssuersValid struct{}

func (i *AllIssuersValid) Validate(creator view2.Identity, tokenType string) error {
	return nil
}

//go:generate counterfeiter -o mock/validator.go -fake-name Validator . Validator

type Validator interface {
	UnmarshallAndVerify(ledger token.Ledger, binding string, raw []byte) ([]interface{}, error)
}

//go:generate counterfeiter -o mock/public_parameters_manager.go -fake-name PublicParametersManager . PublicParametersManager

type PublicParametersManager interface {
}

type TokenChaincode struct {
	initOnce                sync.Once
	LogLevel                string
	Validator               Validator
	PublicParametersManager PublicParametersManager

	PPDigest             []byte
	TokenServicesFactory func([]byte) (PublicParametersManager, Validator, error)

	MetricsEnabled bool
	MetricsServer  string
	MetricsLock    sync.Mutex
	MetricsAgent   Agent
}

func (cc *TokenChaincode) Init(stub shim.ChaincodeStubInterface) pb.Response {
	logger.Infof("init token chaincode...")

	params := cc.ReadParamsFromFile()
	if params == "" {
		if len(Params) == 0 {
			args := stub.GetArgs()
			// args[0] public parameters
			if len(args) != 2 {
				return shim.Error("length of provided arguments != 2")
			}

			if string(args[0]) != "init" {
				return shim.Error("expected init function")
			}
			params = string(args[1])
		} else {
			params = Params
		}
	}

	ppRaw, err := base64.StdEncoding.DecodeString(params)
	if err != nil {
		return shim.Error("failed to decode public parameters: " + err.Error())
	}

	issuingValidator := &AllIssuersValid{}
	rwset := &rwsWrapper{stub: stub}
	w := translator.New(issuingValidator, "", rwset, "")
	action := &SetupAction{
		SetupParameters: ppRaw,
	}

	err = w.Write(action)
	if err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (cc *TokenChaincode) Invoke(stub shim.ChaincodeStubInterface) (res pb.Response) {
	defer func() {
		if r := recover(); r != nil {
			logger.Errorf("invoke triggered panic: %s\n%s\n", r, debug.Stack())
			res = shim.Error(fmt.Sprintf("failed responding [%s]", r))
		} else {
			logger.Infof("execution terminated with status [%d]", res.Status)
		}
	}()

	args := stub.GetArgs()
	switch l := len(args); l {
	case 0:
		return shim.Error("missing parameters")
	default:
		agent, err := cc.NewMetricsAgent(string(args[0]))
		if err != nil {
			return shim.Error(err.Error())
		}
		agent.EmitKey(0, "tcc", "start", "TokenChaincodeInvoke"+string(args[0]), stub.GetTxID())
		defer agent.EmitKey(0, "tcc", "end", "TokenChaincodeInvoke"+string(args[0]), stub.GetTxID())

		logger.Infof("running function [%s]", string(args[0]))
		switch f := string(args[0]); f {
		case InvokeFunction:
			if len(args) != 1 {
				return shim.Error("empty token request")
			}
			// extract token request from transient
			t, err := stub.GetTransient()
			if err != nil {
				return shim.Error("failed getting transient")
			}
			tokenRequest, ok := t["token_request"]
			if !ok {
				return shim.Error("failed getting token request, entry not found")
			}
			return cc.ProcessRequest(tokenRequest, stub)
		case QueryPublicParamsFunction:
			return cc.QueryPublicParams(stub)
		case QueryTokensFunctions:
			if len(args) != 2 {
				return shim.Error("request to retrieve tokens is empty")
			}
			return cc.QueryTokens(args[1], stub)
		default:
			return shim.Error(fmt.Sprintf("function not [%s] recognized", f))
		}
	}
}

func (cc *TokenChaincode) ReadParamsFromFile() string {
	publicParamsPath := os.Getenv(PublicParamsPathVarEnv)
	if publicParamsPath == "" {
		fmt.Println("no PUBLIC_PARAMS_FILE_PATH provided")
		return ""
	}

	fmt.Println("reading " + publicParamsPath + " ...")
	paramsAsBytes, err := ioutil.ReadFile(publicParamsPath)
	if err != nil {
		fmt.Println(fmt.Sprintf(
			"unable to read file %s (%s). continue looking pub params from init args or cc", publicParamsPath, err.Error(),
		))
		return ""
	}

	return base64.StdEncoding.EncodeToString(paramsAsBytes)
}

func (cc *TokenChaincode) GetPublicParametersManager(stub shim.ChaincodeStubInterface) (PublicParametersManager, error) {
	if err := cc.Initialize(stub); err != nil {
		return nil, err
	}
	return cc.PublicParametersManager, nil
}

func (cc *TokenChaincode) GetValidator(stub shim.ChaincodeStubInterface) (Validator, error) {
	var firstInitError error
	cc.initOnce.Do(func() {
		if err := cc.Initialize(stub); err != nil {
			firstInitError = err
		}
	})

	if firstInitError != nil {
		return nil, firstInitError
	}
	return cc.Validator, nil
}

func (cc *TokenChaincode) Initialize(stub shim.ChaincodeStubInterface) error {
	logger.Infof("reading public parameters...")

	rwset := &rwsWrapper{stub: stub}
	issuingValidator := &AllIssuersValid{}
	w := translator.New(issuingValidator, stub.GetTxID(), rwset, "")
	ppRaw, err := w.ReadSetupParameters()
	if err != nil {
		return errors.Wrapf(err, "failed to retrieve public parameters")
	}
	logger.Infof("public parameters read [%d]", len(ppRaw))
	if len(ppRaw) == 0 {
		return errors.Errorf("public parameters are not initiliazed yet")
	}
	hash := sha256.New()
	n, err := hash.Write(ppRaw)
	if n != len(ppRaw) {
		return errors.New("failed hashing public parameters, bytes not consumed")
	}
	if err != nil {
		return errors.Wrap(err, "failed hashing public parameters")
	}
	digest := hash.Sum(nil)
	if len(cc.PPDigest) != 0 && cc.Validator != nil && bytes.Equal(digest, cc.PPDigest) {
		logger.Infof("no need instantiate public parameter manager and validator, already set")
		return nil
	}

	logger.Infof("instantiate public parameter manager and validator...")
	ppm, validator, err := cc.TokenServicesFactory(ppRaw)
	logger.Infof("instantiate public parameter manager and validator done with err [%v]", err)
	if err != nil {
		return errors.Wrap(err, "failed to instantiate public parameter manager and validator")
	}
	cc.PublicParametersManager = ppm
	cc.Validator = validator
	cc.PPDigest = digest

	return nil
}

func (cc *TokenChaincode) ProcessRequest(raw []byte, stub shim.ChaincodeStubInterface) pb.Response {
	cc.MetricsAgent.EmitKey(0, "tcc", "start", "TokenChaincodeProcessRequestGetValidator", stub.GetTxID())
	validator, err := cc.GetValidator(stub)
	cc.MetricsAgent.EmitKey(0, "tcc", "end", "TokenChaincodeProcessRequestGetValidator", stub.GetTxID())
	if err != nil {
		return shim.Error(err.Error())
	}

	// Verify
	cc.MetricsAgent.EmitKey(0, "tcc", "start", "TokenChaincodeProcessRequestUnmarshallAndVerify", stub.GetTxID())
	actions, err := validator.UnmarshallAndVerify(stub, stub.GetTxID(), raw)
	if err != nil {
		return shim.Error("failed to verify token request: " + err.Error())
	}
	cc.MetricsAgent.EmitKey(0, "tcc", "end", "TokenChaincodeProcessRequestUnmarshallAndVerify", stub.GetTxID())

	// Write
	cc.MetricsAgent.EmitKey(0, "tcc", "start", "TokenChaincodeProcessRequestWrite", stub.GetTxID())
	rwset := &rwsWrapper{stub: stub}
	issuingValidator := &AllIssuersValid{}
	w := translator.New(issuingValidator, stub.GetTxID(), rwset, "")
	for _, action := range actions {
		err = w.Write(action)
		if err != nil {
			return shim.Error("failed to write token action: " + err.Error())
		}
	}
	err = w.CommitTokenRequest(raw, false)
	if err != nil {
		return shim.Error("failed to write token request:" + err.Error())
	}
	cc.MetricsAgent.EmitKey(0, "tcc", "end", "TokenChaincodeProcessRequest", stub.GetTxID())

	return shim.Success(nil)
}

func (cc *TokenChaincode) QueryPublicParams(stub shim.ChaincodeStubInterface) pb.Response {
	rwset := &rwsWrapper{stub: stub}
	issuingValidator := &AllIssuersValid{}
	w := translator.New(issuingValidator, stub.GetTxID(), rwset, "")
	raw, err := w.ReadSetupParameters()
	if err != nil {
		shim.Error("failed to retrieve public parameters: " + err.Error())
	}
	if len(raw) == 0 {
		return shim.Error("need to initialize public parameters")
	}
	return shim.Success(raw)
}

func (cc *TokenChaincode) QueryTokens(idsRaw []byte, stub shim.ChaincodeStubInterface) pb.Response {
	var ids []*token2.ID
	if err := json.Unmarshal(idsRaw, &ids); err != nil {
		logger.Errorf("failed unmarshalling tokens ids: [%s]", err)
		return shim.Error(err.Error())
	}

	logger.Debugf("query tokens [%v]...", ids)

	rwset := &rwsWrapper{stub: stub}
	issuingValidator := &AllIssuersValid{}
	w := translator.New(issuingValidator, stub.GetTxID(), rwset, "")
	res, err := w.QueryTokens(ids)
	if err != nil {
		logger.Errorf("failed query tokens [%v]: [%s]", ids, err)
		return shim.Error(fmt.Sprintf("failed query tokens [%v]: [%s]", ids, err))
	}
	raw, err := json.Marshal(res)
	if err != nil {
		logger.Errorf("failed marshalling tokens: [%s]", err)
		return shim.Error(fmt.Sprintf("failed marshalling tokens: [%s]", err))
	}
	return shim.Success(raw)
}

func (cc *TokenChaincode) NewMetricsAgent(id string) (Agent, error) {
	cc.MetricsLock.Lock()
	defer cc.MetricsLock.Unlock()

	if cc.MetricsAgent != nil {
		return cc.MetricsAgent, nil
	}

	if !cc.MetricsEnabled {
		cc.MetricsAgent = metrics.NewNullAgent()
		return cc.MetricsAgent, nil
	}

	var err error
	cc.MetricsAgent, err = metrics.NewStatsdAgent(
		tracing.Host(id),
		tracing.StatsDSink(cc.MetricsServer),
	)
	if err != nil {
		return nil, err
	}
	return cc.MetricsAgent, nil
}
