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
	"go.opentelemetry.io/otel/trace"
)

// SpendRequest is the request to spend a token
type SpendRequest struct {
	Token *token.UnspentToken
}

func ReceiveSpendRequest(context view.Context, opts ...ttx.TxOption) (*SpendRequest, error) {
	logger.Debugf("receive a new spendRequest...")
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

// ReceiveSpendRequestView receives a SpendRequest from the context's session.
type ReceiveSpendRequestView struct{}

func NewReceiveSpendRequestView() *ReceiveSpendRequestView {
	return &ReceiveSpendRequestView{}
}

func (f *ReceiveSpendRequestView) Call(context view.Context) (interface{}, error) {
	span := trace.SpanFromContext(context.Context())
	span.AddEvent("start_receive_spendRequest_view")
	defer span.AddEvent("end_receive_spendRequest_view")
	tx := &SpendRequest{}
	jsonSession := session.JSON(context)
	err := jsonSession.ReceiveWithTimeout(tx, time.Minute*4)
	if err != nil {
		span.RecordError(err)
	}
	span.AddEvent("receive_tx")
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

	ok, identities, err := multisig.Unwrap(unspentToken.Owner)
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
	span := trace.SpanFromContext(context.Context())

	// send Transaction to each party and wait for their responses
	request := &SpendRequest{Token: c.unspentToken}
	requestRaw, err := request.Bytes()
	if err != nil {
		return nil, err
	}

	answerChannel := make(chan *answer, len(c.parties))
	span.AddEvent("request_spend")
	logger.Debugf("notify [%d] parties about request [%v]", len(c.parties), request)
	counter := 0
	tms := token2.GetManagementService(context, token2.WithTMSID(c.options.TMSID()))
	if tms == nil {
		return nil, errors.Errorf("failed getting TMS for [%s]", c.options.TMSID())
	}
	areMe := tms.SigService().AreMe(c.parties...)
	for _, party := range c.parties {
		logger.Debugf("notify party [%s] about request...", party)
		if slices.Contains(areMe, party.UniqueID()) {
			// it is me, skip
			logger.Debugf("notify party [%s] about request, it is me, skipping...", party)
			continue
		}
		go c.collectSpendRequestAnswers(context, party, requestRaw, answerChannel)
		counter++
	}

	for i := 0; i < counter; i++ {
		span.AddEvent("wait_for_answer")
		// TODO: put a timeout
		a := <-answerChannel
		span.AddEvent("received_answer")
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
	raw []byte,
	answerChan chan *answer) {
	defer logger.Debugf("received response for from [%v]", party)

	initiator := context.Initiator()
	if c.options.Initiator != nil {
		initiator = c.options.Initiator
	}
	s, err := session.NewJSON(context, initiator, party)
	if err != nil {
		answerChan <- &answer{
			err:   errors.Wrapf(err, "failed to create session with [%s]", party),
			party: party,
		}
		return
	}

	// Wait to receive a Transaction back
	logger.Debugf("send request to [%v]", party)
	err = s.SendRaw(context.Context(), raw)
	if err != nil {
		answerChan <- &answer{
			err:   errors.Wrapf(err, "failed to send request to [%s]", party),
			party: party,
		}
		return
	}
	response := &SpendResponse{}
	if err := s.Receive(response); err != nil {
		answerChan <- &answer{
			err:   errors.Wrapf(err, "failed to receive response from [%s]", party),
			party: party,
		}
		return
	}
	logger.Debugf("received response from [%v]: [%v]", party, response.Err)

	answerChan <- &answer{response: response, party: party}
}

// EndorseSpendView endorses a SpendRequest
type EndorseSpendView struct {
	request *SpendRequest
}

func NewEndorseSpendView(request *SpendRequest) *EndorseSpendView {
	return &EndorseSpendView{request: request}
}

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

func (a *EndorseSpendView) Call(context view.Context) (interface{}, error) {
	// - send back the response
	if err := session.JSON(context).Send(&SpendResponse{}); err != nil {
		return nil, errors.Wrap(err, "failed to send response")
	}

	logger.Debugf("endorse spend response sent")
	// At some point, the recipient receives the token transaction that in the meantime has been assembled
	tx, err := ttx.ReceiveTransaction(context)
	if err != nil {
		return nil, errors.Wrap(err, "failed to receive transaction")
	}
	logger.Debugf("multisig tx received with id [%s]", tx.ID())

	// TODO: check tx matches request

	// If everything is fine, the recipient accepts and sends back her signature.
	// Notice that, a signature from the recipient might or might not be required to make the transaction valid.
	// This depends on the driver implementation.
	logger.Debugf("endorse multisig tx received with id [%s]", tx.ID())
	_, err = context.RunView(ttx.NewEndorseView(tx))
	if err != nil {
		return nil, errors.Wrap(err, "failed to accept transaction")
	}

	logger.Debugf("endorse multisig tx received with id [%s] done", tx.ID())

	return tx, nil
}
