package views

import (
	"encoding/json"
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/assert"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
)

type RegisterOwnerWallet struct {
	ID   string
	Type string
	Path string
}

type RegisterOwnerWalletView struct {
	*RegisterOwnerWallet
}

// Call the view to register a new owner wallet
func (view *RegisterOwnerWalletView) Call(context view.Context) (interface{}, error) {
	fmt.Printf("Registering owner wallet %s:%s:%s\n", view.ID, view.Type, view.Path)
	err := token.GetManagementService(context).WalletManager().RegisterOwnerWallet(
		view.ID, view.Type, view.Path,
	)
	fmt.Printf("Registering owner wallet %s:%s:%s done, checking error...\n", view.ID, view.Type, view.Path)
	assert.NoError(err, "Failed to register owner wallet")
	fmt.Printf("no error \n")

	return nil, nil
}

type RegisterOwnerWalletViewFactory struct{}

func (p *RegisterOwnerWalletViewFactory) NewView(in []byte) (view.View, error) {
	f := &RegisterOwnerWalletView{RegisterOwnerWallet: &RegisterOwnerWallet{}}
	err := json.Unmarshal(in, f.RegisterOwnerWallet)
	assert.NoError(err, "failed unmarshalling input")
	return f, nil
}
