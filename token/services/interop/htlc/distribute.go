/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package htlc

import (
	"encoding/json"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/session"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

// Terms contains the details of the htlc to be examined
type Terms struct {
	ReclamationDeadline time.Duration
	TMSID1              token.TMSID
	Type1               string
	Amount1             uint64
	TMSID2              token.TMSID
	Type2               string
	Amount2             uint64
}

// Bytes serializes the terms
func (t *Terms) Bytes() ([]byte, error) {
	return json.Marshal(t)
}

// FromBytes unmarshals terms
func (t *Terms) FromBytes(raw []byte) error {
	return json.Unmarshal(raw, t)
}

// Validate checks the terms
func (t *Terms) Validate() error {
	if t.ReclamationDeadline <= 0 {
		return errors.New("reclamation deadline should be larger than zero")
	}
	if t.Type1 == "" || t.Type2 == "" {
		return errors.New("types should be set")
	}
	if t.Amount1 <= 0 || t.Amount2 <= 0 {
		return errors.New("amounts should be larger than zero")
	}
	return nil
}

// DistributeTermsView holds the terms and the recipient identity to be used by the view
type DistributeTermsView struct {
	recipient view.Identity
	terms     *Terms
}

// NewDistributeTermsView creates a view which distributes the terms to the recipient
func NewDistributeTermsView(recipient view.Identity, terms *Terms) *DistributeTermsView {
	return &DistributeTermsView{
		recipient: recipient,
		terms:     terms,
	}
}

func (v *DistributeTermsView) Call(context view.Context) (interface{}, error) {
	session, err := context.GetSession(context.Initiator(), v.recipient)
	if err != nil {
		return nil, err
	}
	termsRaw, err := v.terms.Bytes()
	if err != nil {
		return nil, errors.Wrapf(err, "failed marshalling terms")
	}
	err = session.Send(termsRaw)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

type termsReceiverView struct{}

// ReceiveTerms runs the termsReceiverView and returns the received terms
func ReceiveTerms(context view.Context) (*Terms, error) {
	terms, err := context.RunView(&termsReceiverView{})
	if err != nil {
		return nil, err
	}
	return terms.(*Terms), nil
}

func (v *termsReceiverView) Call(context view.Context) (interface{}, error) {
	_, payload, err := session.ReadFirstMessage(context)
	if err != nil {
		return nil, err
	}
	terms := &Terms{}
	if err := terms.FromBytes(payload); err != nil {
		return nil, errors.Wrapf(err, "failed unmarshalling terms")
	}
	return terms, nil
}
