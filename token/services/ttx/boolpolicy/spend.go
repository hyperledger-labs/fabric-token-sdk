/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package boolpolicy provides a spend-coordination protocol for policy identity tokens.
// For OR policies a single co-owner can spend unilaterally; for AND policies all
// co-owners must endorse.  The RequestSpendView / EndorseSpendView pair mirrors the
// multisig spend protocol and is reused for the AND case.
package boolpolicy

import (
	"slices"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/boolpolicy"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// SpendRequest carries a policy token selected for spending to co-owners.
type SpendRequest struct {
	Token *token.UnspentToken
}

// Bytes serialises the request.
func (r *SpendRequest) Bytes() ([]byte, error) {
	return json.Marshal(r)
}

// String returns a brief description.
func (r *SpendRequest) String() string {
	if r.Token == nil {
		return ""
	}

	return r.Token.String()
}

// ReceiveSpendRequest receives an incoming SpendRequest on the current session.
func ReceiveSpendRequest(context view.Context) (*SpendRequest, error) {
	logger.DebugfContext(context.Context(), "receive a new policy spendRequest...")
	requestBoxed, err := context.RunView(NewReceiveSpendRequestView(), view.WithSameContext())
	if err != nil {
		return nil, err
	}
	request, ok := requestBoxed.(*SpendRequest)
	if !ok {
		return nil, errors.Errorf("received spendRequest of wrong type [%T]", request)
	}

	return request, nil
}

// ReceiveSpendRequestView is the responder-side view that reads a SpendRequest.
type ReceiveSpendRequestView struct{}

// NewReceiveSpendRequestView returns a new ReceiveSpendRequestView.
func NewReceiveSpendRequestView() *ReceiveSpendRequestView {
	return &ReceiveSpendRequestView{}
}

// Call implements view.View.
func (f *ReceiveSpendRequestView) Call(context view.Context) (interface{}, error) {
	tx := &SpendRequest{}
	if err := session.JSON(context).ReceiveWithTimeout(tx, time.Minute*4); err != nil {
		logger.ErrorfContext(context.Context(), "failed receiving request: %s", err)

		return nil, err
	}

	return tx, nil
}

// SpendResponse is the ACK returned by a co-owner after receiving a SpendRequest.
type SpendResponse struct {
	Err error
}

type answer struct {
	response *SpendResponse
	err      error
	party    view.Identity
}

// RequestSpendView sends a SpendRequest to all co-owners of a policy token and
// collects their acknowledgements.  This is needed for AND policies; OR-policy
// spends can skip this step.
type RequestSpendView struct {
	unspentToken *token.UnspentToken
	parties      []view.Identity
	options      *token2.ServiceOptions

	err error
}

// NewRequestSpendView creates a RequestSpendView for the given policy token.
func NewRequestSpendView(unspentToken *token.UnspentToken, opts ...token2.ServiceOption) *RequestSpendView {
	if unspentToken == nil {
		return &RequestSpendView{err: errors.Errorf("unspentToken is nil")}
	}
	serviceOptions, err := token2.CompileServiceOptions(opts...)
	if err != nil {
		return &RequestSpendView{err: errors.Wrap(err, "failed to compile service options")}
	}
	pi, ok, err := boolpolicy.Unwrap(unspentToken.Owner)
	if err != nil {
		return &RequestSpendView{err: errors.Wrap(err, "failed to unwrap policy identity")}
	}
	if !ok {
		return &RequestSpendView{err: errors.Errorf("token is not a policy identity")}
	}
	parties := make([]view.Identity, len(pi.Identities))
	for i, b := range pi.Identities {
		parties[i] = b
	}

	return &RequestSpendView{
		unspentToken: unspentToken,
		parties:      parties,
		options:      serviceOptions,
	}
}

// Call implements view.View.
func (c *RequestSpendView) Call(context view.Context) (interface{}, error) {
	if c.err != nil {
		return nil, c.err
	}
	request := &SpendRequest{Token: c.unspentToken}
	requestRaw, err := request.Bytes()
	if err != nil {
		return nil, err
	}
	tms, err := token2.GetManagementService(context, token2.WithTMSID(c.options.TMSID()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting TMS for [%s]", c.options.TMSID())
	}
	areMe := tms.SigService().AreMe(context.Context(), c.parties...)
	answerChannel := make(chan *answer, len(c.parties))
	counter := 0
	for _, party := range c.parties {
		if slices.Contains(areMe, party.UniqueID()) {
			continue
		}
		go c.collectAnswers(context, party, requestRaw, answerChannel)
		counter++
	}
	for range counter {
		a := <-answerChannel
		if a.err != nil {
			return nil, errors.Wrapf(a.err, "failure from [%s]", a.party)
		}
		if a.response.Err != nil {
			return nil, errors.Wrapf(a.response.Err, "failure from [%s]", a.party)
		}
	}

	return nil, nil
}

func (c *RequestSpendView) collectAnswers(context view.Context, party view.Identity, raw []byte, ch chan *answer) {
	defer logger.DebugfContext(context.Context(), "received response from [%v]", party)

	backendSession, err := context.GetSession(c, party, context.Initiator())
	if err != nil {
		ch <- &answer{err: errors.Wrapf(err, "failed to create session with [%s]", party), party: party}

		return
	}
	s := session.NewFromSession(context, backendSession)
	if err = s.SendRaw(context.Context(), raw); err != nil {
		ch <- &answer{err: errors.Wrapf(err, "failed to send request to [%s]", party), party: party}

		return
	}
	response := &SpendResponse{}
	if err := s.Receive(response); err != nil {
		ch <- &answer{err: errors.Wrapf(err, "failed to receive response from [%s]", party), party: party}

		return
	}
	ch <- &answer{response: response, party: party}
}

// EndorseSpendView is the co-owner's view: it ACKs the spend request and then
// endorses the assembled transaction.
type EndorseSpendView struct {
	request *SpendRequest
}

// NewEndorseSpendView returns a new EndorseSpendView.
func NewEndorseSpendView(request *SpendRequest) *EndorseSpendView {
	return &EndorseSpendView{request: request}
}

// EndorseSpend is a convenience wrapper that runs NewEndorseSpendView.
func EndorseSpend(context view.Context, request *SpendRequest) (*Transaction, error) {
	resultBoxed, err := context.RunView(NewEndorseSpendView(request))
	if err != nil {
		return nil, errors.Wrap(err, "failed to approve spend")
	}
	result, ok := resultBoxed.(*ttx.Transaction)
	if !ok {
		return nil, errors.Errorf("received result of wrong type [%T]", result)
	}

	return &Transaction{Transaction: result}, nil
}

// Call implements view.View.
func (a *EndorseSpendView) Call(context view.Context) (interface{}, error) {
	if err := session.JSON(context).Send(&SpendResponse{}); err != nil {
		return nil, errors.Wrap(err, "failed to send spend response")
	}
	tx, err := ttx.ReceiveTransaction(context)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive transaction")
	}
	if _, err = context.RunView(ttx.NewEndorseView(tx)); err != nil {
		return nil, errors.Wrap(err, "failed to endorse transaction")
	}

	return tx, nil
}
