/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	view3 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

// WithType returns a list token option that filter by the passed token type.
// If the passed token type is the empty string, all token types are selected.
func WithType(tokenType string) token.ListTokensOption {
	return func(o *token.ListTokensOptions) error {
		o.TokenType = tokenType
		return nil
	}
}

// MyWallet returns the default wallet, nil if not found.
func MyWallet(sp view2.ServiceProvider, opts ...token.ServiceOption) *token.OwnerWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().OwnerWallet("")
	if w == nil {
		return nil
	}
	return w
}

// MyWalletFromTx returns the default wallet for the tuple (network, channel, namespace) as identified by the passed
// transaction. Returns nil if no wallet is found.
func MyWalletFromTx(sp view2.ServiceProvider, tx *Transaction) *token.OwnerWallet {
	w := token.GetManagementService(
		sp,
		token.WithNetwork(tx.Network()),
		token.WithChannel(tx.Channel()),
		token.WithNamespace(tx.Namespace()),
	).WalletManager().OwnerWallet("")
	if w == nil {
		return nil
	}
	return w
}

// GetWallet returns the wallet whose id is the passed id.
// If the passed id is empty, GetWallet has the same behaviour of MyWallet.
// It returns nil, if no wallet is found.
func GetWallet(sp view2.ServiceProvider, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().OwnerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// GetWalletForChannel returns the wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetWalletForChannel has the same behaviour of MyWalletFromTx.
// It returns nil, if no wallet is found.
func GetWalletForChannel(sp view2.ServiceProvider, channel, id string, opts ...token.ServiceOption) *token.OwnerWallet {
	w := token.GetManagementService(sp, append(opts, token.WithChannel(channel))...).WalletManager().OwnerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// MyIssuerWallet returns the default issuer wallet, nil if not found
func MyIssuerWallet(context view.Context, opts ...token.ServiceOption) *token.IssuerWallet {
	w := token.GetManagementService(context, opts...).WalletManager().IssuerWallet("")
	if w == nil {
		return nil
	}
	return w
}

// GetIssuerWallet returns the issuer wallet whose id is the passed id.
// If the passed id is empty, GetIssuerWallet has the same behaviour of MyIssuerWallet.
// It returns nil, if no wallet is found.
func GetIssuerWallet(sp view2.ServiceProvider, id string, opts ...token.ServiceOption) *token.IssuerWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().IssuerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// GetIssuerWalletForChannel returns the issuer wallet whose id is the passed id for the passed channel.
// If the passed id is empty, GetIssuerWalletForChannel has the same behaviour of MyIssuerWallet.
// It returns nil, if no wallet is found.
func GetIssuerWalletForChannel(sp view2.ServiceProvider, channel, id string, opts ...token.ServiceOption) *token.IssuerWallet {
	w := token.GetManagementService(sp, append(opts, token.WithChannel(channel))...).WalletManager().IssuerWallet(id)
	if w == nil {
		return nil
	}
	return w
}

// MyAuditorWallet returns the default auditor wallet, nil if not found.
func MyAuditorWallet(sp view2.ServiceProvider, opts ...token.ServiceOption) *token.AuditorWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().AuditorWallet("")
	if w == nil {
		return nil
	}
	return w
}

// GetAuditorWallet returns the wallet whose id is the passed id.
// If the passed id is empty, GetAuditorWallet has the same behaviour of MyAuditorWallet.
// It returns nil, if no wallet is found.
func GetAuditorWallet(sp view2.ServiceProvider, opts ...token.ServiceOption) *token.AuditorWallet {
	w := token.GetManagementService(sp, opts...).WalletManager().AuditorWallet("")
	if w == nil {
		return nil
	}
	return w
}

type StreamExternalWalletSignRequest struct {
	Party   view.Identity
	Message []byte
}

type StreamExternalWalletSignResponse struct {
	Sigma []byte
}

type StreamExternalWalletSignerServer struct {
	stream view3.Stream
}

func NewStreamExternalWalletSignerServer(stream view3.Stream) *StreamExternalWalletSignerServer {
	return &StreamExternalWalletSignerServer{stream: stream}
}

func (s *StreamExternalWalletSignerServer) Sign(party view.Identity, message []byte) ([]byte, error) {
	logger.Info("send sign request for party [%s]", party)
	if err := s.stream.Send(&StreamExternalWalletSignRequest{
		Party:   party,
		Message: message,
	}); err != nil {
		return nil, err
	}
	logger.Info("receive response, party [%s]", party)
	response := &StreamExternalWalletSignResponse{}
	if err := s.stream.Recv(response); err != nil {
		return nil, err
	}
	return response.Sigma, nil
}

type SignerProvider interface {
	GetSigner(party view.Identity) (token.Signer, error)
}

type StreamExternalWalletSignerClient struct {
	sp               SignerProvider
	stream           view3.Stream
	expectedRequests int
}

func NewStreamExternalWalletSignerClient(sp SignerProvider, stream view3.Stream, expectedRequests int) *StreamExternalWalletSignerClient {
	return &StreamExternalWalletSignerClient{
		sp:               sp,
		stream:           stream,
		expectedRequests: expectedRequests,
	}
}

func (s *StreamExternalWalletSignerClient) Respond() error {
	for i := 0; i < s.expectedRequests; i++ {
		logger.Infof("process signature request [%d]", i)
		req := &StreamExternalWalletSignRequest{}
		if err := s.stream.Recv(req); err != nil {
			return errors.Wrapf(err, "failed to receive signature request [%d]", i)
		}
		signer, err := s.sp.GetSigner(req.Party)
		if err != nil {
			return errors.Wrapf(err, "failed to get signer for party [%s]", req.Party)
		}
		sigma, err := signer.Sign(req.Message)
		if err != nil {
			return errors.Wrapf(err, "failed to sign, party [%s]", req.Party)
		}
		if err := s.stream.Send(&StreamExternalWalletSignResponse{Sigma: sigma}); err != nil {
			return errors.Wrapf(err, "failed to send back signature, party [%s]", req.Party)
		}
		logger.Infof("process signature request done [%d]", i)

	}
	return nil
}
