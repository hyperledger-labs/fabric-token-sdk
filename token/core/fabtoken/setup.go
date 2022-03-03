/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"encoding/json"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

const Coin = uint64(1000000000)
const MaxMoney = uint64(21000000) * Coin
const PublicParameters = "fabtoken"

type PublicParams struct {
	Label   string
	MTV     uint64
	Auditor []byte
	Issuers [][]byte
}

func NewPublicParamsFromBytes(raw []byte, label string) (*PublicParams, error) {
	pp := &PublicParams{}
	pp.Label = label
	if err := pp.Deserialize(raw); err != nil {
		return nil, errors.Wrap(err, "failed parsing public parameters")
	}
	return pp, nil
}

func (pp *PublicParams) Identifier() string {
	return pp.Label
}

func (pp *PublicParams) TokenDataHiding() bool {
	return false
}

func (pp *PublicParams) CertificationDriver() string {
	return pp.Label
}

func (pp *PublicParams) GraphHiding() bool {
	return false
}

func (pp *PublicParams) MaxTokenValue() uint64 {
	return pp.MTV
}

func (pp *PublicParams) Bytes() ([]byte, error) {
	return json.Marshal(pp)
}

func (pp *PublicParams) Serialize() ([]byte, error) {
	raw, err := json.Marshal(pp)
	if err != nil {
		return nil, err
	}
	return json.Marshal(&driver.SerializedPublicParameters{
		Identifier: pp.Label,
		Raw:        raw,
	})
}

func (pp *PublicParams) Deserialize(raw []byte) error {
	publicParams := &driver.SerializedPublicParameters{}
	if err := json.Unmarshal(raw, publicParams); err != nil {
		return err
	}
	if publicParams.Identifier != pp.Label {
		return errors.Errorf("invalid identifier, expecting 'fabtoken', got [%s]", publicParams.Identifier)
	}
	return json.Unmarshal(publicParams.Raw, pp)
}

func (pp *PublicParams) AuditorIdentity() view.Identity {
	return pp.Auditor
}

func (pp *PublicParams) AddAuditor(auditor view.Identity) {
	pp.Auditor = auditor
}

func (pp *PublicParams) AddIssuer(issuer view.Identity) {
	pp.Issuers = append(pp.Issuers, issuer)
}

func Setup() (*PublicParams, error) {
	return &PublicParams{
		MTV:   MaxMoney,
		Label: PublicParameters,
	}, nil
}
