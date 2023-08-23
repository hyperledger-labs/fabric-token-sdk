/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/json"

	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

type StreamExternalWalletMsgType = int

const (
	_ StreamExternalWalletMsgType = iota
	SigRequest
	SignResponse
	Done
)

type StreamExternalWalletMsg struct {
	Type StreamExternalWalletMsgType
	Raw  []byte
}

func NewStreamExternalWalletMsg(Type StreamExternalWalletMsgType, v interface{}) (*StreamExternalWalletMsg, error) {
	var raw []byte
	if v != nil {
		var err error
		raw, err = json.Marshal(v)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal [%v]", v)
		}
	}
	return &StreamExternalWalletMsg{Type: Type, Raw: raw}, nil
}

type StreamExternalWalletSignRequest struct {
	Party   view.Identity
	Message []byte
}

type StreamExternalWalletSignResponse struct {
	Sigma []byte
}

type StreamExternalWalletSignerServer struct {
	stream view2.Stream
}

func NewStreamExternalWalletSignerServer(stream view2.Stream) *StreamExternalWalletSignerServer {
	return &StreamExternalWalletSignerServer{stream: stream}
}

func (s *StreamExternalWalletSignerServer) Sign(party view.Identity, message []byte) ([]byte, error) {
	logger.Info("send sign request for party [%s]", party)
	msg, err := NewStreamExternalWalletMsg(SigRequest, &StreamExternalWalletSignRequest{
		Party:   party,
		Message: message,
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal sign request message")
	}
	if err := s.stream.Send(msg); err != nil {
		return nil, err
	}
	logger.Info("receive response, party [%s]", party)

	msg = &StreamExternalWalletMsg{}
	if err := s.stream.Recv(msg); err != nil {
		return nil, err
	}
	if msg.Type != SignResponse {
		return nil, errors.Errorf("expected sign response msg, got [%d]", msg.Type)
	}
	response := &StreamExternalWalletSignResponse{}
	if err := json.Unmarshal(msg.Raw, response); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal sign response")
	}
	return response.Sigma, nil
}

func (s *StreamExternalWalletSignerServer) Done() error {
	logger.Info("send done...")
	msg, err := NewStreamExternalWalletMsg(Done, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal sign request message")
	}
	if err := s.stream.Send(msg); err != nil {
		return err
	}
	return nil
}

type SignerProvider interface {
	GetSigner(party view.Identity) (token.Signer, error)
}

type StreamExternalWalletSignerClient struct {
	sp               SignerProvider
	stream           view2.Stream
	expectedRequests int
}

func NewStreamExternalWalletSignerClient(sp SignerProvider, stream view2.Stream, expectedRequests int) *StreamExternalWalletSignerClient {
	return &StreamExternalWalletSignerClient{
		sp:               sp,
		stream:           stream,
		expectedRequests: expectedRequests,
	}
}

func (s *StreamExternalWalletSignerClient) Respond() error {
	i := 0
	for {
		logger.Infof("process signature request [%d]", i)

		msg := &StreamExternalWalletMsg{}
		if err := s.stream.Recv(msg); err != nil {
			return errors.Wrapf(err, "failed to receive signature request [%d]", i)
		}
		stop := false
		switch msg.Type {
		case SigRequest:
			req := &StreamExternalWalletSignRequest{}
			if err := json.Unmarshal(msg.Raw, req); err != nil {
				return errors.Wrapf(err, "failed to get unmarshal msg type SigRequest")
			}
			signer, err := s.sp.GetSigner(req.Party)
			if err != nil {
				return errors.Wrapf(err, "failed to get signer for party [%s]", req.Party)
			}
			sigma, err := signer.Sign(req.Message)
			if err != nil {
				return errors.Wrapf(err, "failed to sign, party [%s]", req.Party)
			}

			msg, err := NewStreamExternalWalletMsg(SignResponse, &StreamExternalWalletSignResponse{
				Sigma: sigma,
			})
			if err != nil {
				return errors.Wrapf(err, "failed to marshal sign request message")
			}
			if err := s.stream.Send(msg); err != nil {
				return errors.Wrapf(err, "failed to send back signature, party [%s]", req.Party)
			}
			logger.Infof("process signature request done [%d]", i)
		case Done:
			logger.Infof("no more signatures required")
			stop = true
		}
		if stop {
			break
		}
		i++
	}
	return nil
}
