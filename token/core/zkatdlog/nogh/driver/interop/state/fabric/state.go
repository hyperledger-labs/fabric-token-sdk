/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
	fabric3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/state/fabric/core"
	"go.uber.org/dig"
)

type Validator struct{}

func (v *Validator) Validate(tokRaw []byte, info *pledge.Info) error {
	// TODO: complete this
	// tok := &token.Token{}
	// err := json.Unmarshal(tokRaw, tok)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to unmarshal token [%s]", common.Hashable(tokRaw))
	// }
	//
	// if tok.Type != info.TokenType {
	// 	return errors.Errorf("type of pledge token does not match type in claim request")
	// }
	// q, err := token.ToQuantity(tok.Quantity, 64)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed converting token quantity [%s]", tok.Quantity)
	// }
	// expectedQ := token.NewQuantityFromUInt64(info.Amount)
	// if expectedQ.Cmp(q) != 0 {
	// 	return errors.Errorf("quantity in pledged token is different from quantity in claim request")
	// }
	// owner, err := identity.UnmarshalTypedIdentity(tok.Owner.Raw)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to unmarshal owner of token [%s]", common.Hashable(tokRaw))
	// }
	// if owner.Type != pledge.ScriptType {
	// 	return err
	// }
	// script := &pledge.Script{}
	// err = json.Unmarshal(owner.Identity, script)
	// if err != nil {
	// 	return errors.Wrapf(err, "failed to unmarshal pledge script [%s]", common.Hashable(tokRaw))
	// }
	// if script.Recipient == nil {
	// 	return errors.Errorf("script in proof encodes invalid recipient")
	// }
	// if !script.Recipient.Equal(info.Script.Recipient) {
	// 	return errors.Errorf("recipient in claim request does not match recipient in proof")
	// }
	// if script.Deadline != info.Script.Deadline {
	// 	return errors.Errorf("deadline in claim request does not match deadline in proof")
	// }
	// if script.DestinationNetwork != info.Script.DestinationNetwork {
	// 	return errors.Errorf("destination network in claim request does not match destination network in proof [%s vs.%s]", info.Script.DestinationNetwork, script.DestinationNetwork)
	// }

	return nil
}

func NewStateDriver(in struct {
	dig.In
	FNSProvider   *fabric.NetworkServiceProvider
	RelayProvider fabric3.RelayProvider
	VaultStore    *pledge.VaultStore
}) fabric3.NamedStateDriver {
	return fabric3.NamedStateDriver{
		Name: crypto.DLogPublicParameters,
		Driver: core.NewStateDriver(
			logging.MustGetLogger("token-sdk.core.zkatdlog"),
			in.FNSProvider,
			in.RelayProvider,
			in.VaultStore,
			&Validator{},
		),
	}
}
