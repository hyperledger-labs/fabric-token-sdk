/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package multisig

import (
	"slices"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/multisig"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils/json/session"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

// SpendRequest is the request to spend a token
type SpendRequest struct {
	Token *token.UnspentToken
}

func ReceiveSpendRequest(context view.Context, opts ...ttx.TxOption) (*SpendRequest, error) {
	logger.DebugfContext(context.Context(), "receive a new spendRequest...")
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

func (r *SpendRequest) Bytes() ([]byte, error) {
	return json.Marshal(r)
}

func (r *SpendRequest) String() string {
	if r.Token == nil {
		return ""
	}

	return r.Token.String()
}

// ReceiveSpendRequestView receives a SpendRequest from the context's session.
type ReceiveSpendRequestView struct{}

func NewReceiveSpendRequestView() *ReceiveSpendRequestView {
	return &ReceiveSpendRequestView{}
}

func (f *ReceiveSpendRequestView) Call(context view.Context) (interface{}, error) {
	tx := &SpendRequest{}
	s := session.JSON(context)
	if err := session.ReceiveTypedWithTimeout(s, session.TypeSpendRequest, tx, time.Minute*4); err != nil {
		logger.ErrorfContext(context.Context(), "failed receiving request: %s", err)

		return nil, err
	}

	return tx, nil
}

// SpendResponse is the response to a SpendRequest
type SpendResponse struct {
	Err error
}

type answer struct {
	response *SpendResponse
	err      error
	party    view.Identity
}

// RequestSpendView sends a SpendRequest to all parties and waits for their responses
type RequestSpendView struct {
	unspentToken *token.UnspentToken
	parties      []view.Identity
	options      *token2.ServiceOptions

	err     error
	timeout time.Duration
}

func NewRequestSpendView(unspentToken *token.UnspentToken, opts ...token2.ServiceOption) *RequestSpendView {
	if unspentToken == nil {
		return &RequestSpendView{err: errors.Errorf("unspentToken is nil")}
	}

	serviceOptions, err := token2.CompileServiceOptions(opts...)
	if err != nil {
		return &RequestSpendView{err: errors.Wrap(err, "failed to compile service options")}
	}

	identities, ok, err := multisig.Unwrap(unspentToken.Owner)
	if err != nil {
		return &RequestSpendView{err: errors.Wrap(err, "failed to unwrap identities")}
	}
	if !ok {
		return &RequestSpendView{err: errors.Errorf("unwrapping failed")}
	}

	return &RequestSpendView{
		unspentToken: unspentToken,
		parties:      identities,
		options:      serviceOptions,
	}
}

func (c *RequestSpendView) Call(context view.Context) (interface{}, error) {
	if c.err != nil {
		return nil, c.err
	}

	// send Transaction to each party and wait for their responses
	request := &SpendRequest{Token: c.unspentToken}

	answerChannel := make(chan *answer, len(c.parties))
	logger.DebugfContext(context.Context(), "Notify %d parties about request", len(c.parties))
	logger.DebugfContext(context.Context(), "Request [%v]", len(c.parties), request)
	counter := 0
	tms, err := token2.GetManagementService(context, token2.WithTMSID(c.options.TMSID()))
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting TMS for [%s]", c.options.TMSID())
	}
	areMe := tms.SigService().AreMe(context.Context(), c.parties...)
	for _, party := range c.parties {
		logger.DebugfContext(context.Context(), "notify party [%s] about request...", party)
		if slices.Contains(areMe, party.UniqueID()) {
			// it is me, skip
			logger.DebugfContext(context.Context(), "notify party [%s] about request, it is me, skipping...", party)

			continue
		}
		go c.collectSpendRequestAnswers(context, party, request, answerChannel)
		counter++
	}

	for range counter {
		logger.DebugfContext(context.Context(), "Wait for answer")
		// TODO: put a timeout
		a := <-answerChannel
		logger.DebugfContext(context.Context(), "Received answer")
		if a.err != nil {
			return nil, errors.Wrapf(a.err, "got failure [%s] from [%s]", a.party.String(), a.err)
		}
		if a.response.Err != nil {
			return nil, errors.Wrapf(a.response.Err, "got failure [%s] from [%s]", a.party.String(), a.response.Err)
		}
	}

	return nil, nil
}

func (c *RequestSpendView) WithTimeout(timeout time.Duration) *RequestSpendView {
	c.timeout = timeout

	return c
}

func (c *RequestSpendView) collectSpendRequestAnswers(
	context view.Context,
	party view.Identity,
	request *SpendRequest,
	answerChan chan *answer) {
	defer logger.DebugfContext(context.Context(), "received response for from [%v]", party)

	backendSession, err := context.GetSession(c, party, context.Initiator())
	if err != nil {
		answerChan <- &answer{
			err:   errors.Wrapf(err, "failed to create session with [%s]", party),
			party: party,
		}

		return
	}
	s := session.NewFromSession(context, backendSession)

	logger.DebugfContext(context.Context(), "send request to [%v]", party)
	err = session.SendTyped(s, context.Context(), request, session.TypeSpendRequest)
	if err != nil {
		answerChan <- &answer{
			err:   errors.Wrapf(err, "failed to send request to [%s]", party),
			party: party,
		}

		return
	}
	response := &SpendResponse{}
	if err := session.ReceiveTyped(s, session.TypeSpendResponse, response); err != nil {
		answerChan <- &answer{
			err:   errors.Wrapf(err, "failed to receive response from [%s]", party),
			party: party,
		}

		return
	}
	logger.DebugfContext(context.Context(), "received response from [%v]: [%v]", party, response.Err)

	answerChan <- &answer{response: response, party: party}
}

// ReceiveSpendTxView is the co-owner's view: it ACKs the SpendRequest and
// returns the assembled transaction received from the initiator without
// endorsing it. The caller is responsible for inspecting the transaction
// (e.g. confirming it consumes the expected token and does not include
// other tokens owned by this node) and, if the checks pass, running
// ttx.NewEndorseView(tx) to produce the signature.
//
// Splitting receive from endorse lets the application decide which
// business-logic checks to apply rather than baking a fixed policy into
// the library.
type ReceiveSpendTxView struct {
	request *SpendRequest
}

// NewReceiveSpendTxView returns a new ReceiveSpendTxView for the given request.
func NewReceiveSpendTxView(request *SpendRequest) *ReceiveSpendTxView {
	return &ReceiveSpendTxView{request: request}
}

// ReceiveSpendTx is a convenience wrapper that runs ReceiveSpendTxView and
// returns the unsigned spend transaction so the caller can inspect it
// before deciding whether to endorse.
func ReceiveSpendTx(context view.Context, request *SpendRequest) (*Transaction, error) {
	resultBoxed, err := context.RunView(NewReceiveSpendTxView(request))
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive spend transaction")
	}
	result, ok := resultBoxed.(*ttx.Transaction)
	if !ok {
		return nil, errors.Errorf("received result of wrong type [%T]", result)
	}

	return &Transaction{Transaction: result}, nil
}

// Call implements view.View. It sends the SpendResponse ACK, receives the
// assembled transaction, and returns it without endorsing. Endorsement is
// the caller's responsibility once any business-logic checks pass.
func (a *ReceiveSpendTxView) Call(context view.Context) (interface{}, error) {
	s := session.JSON(context)
	if err := session.SendTyped(s, context.Context(), &SpendResponse{}, session.TypeSpendResponse); err != nil {
		return nil, errors.Wrap(err, "failed to send response")
	}
	logger.DebugfContext(context.Context(), "spend response sent")

	tx, err := ttx.ReceiveTransaction(context)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive transaction")
	}
	logger.DebugfContext(context.Context(), "multisig tx received with id [%s]", tx.ID())

	return tx, nil
}
