/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package processor

const (
	AddToken    = "store-token"
	DeleteToken = "delete-token"
	UpdateToken = "update-token"
)

type TokenProcessorEvent struct {
	topic   string
	message TokenMessage
}

func NewTokenProcessorEvent(topic string, message *TokenMessage) *TokenProcessorEvent {
	return &TokenProcessorEvent{
		topic:   topic,
		message: *message,
	}
}

type TokenMessage struct {
	WalletID  string
	TokenType string
	TxID      string
	Index     uint64
}

func (t *TokenProcessorEvent) Topic() string {
	return t.topic
}

func (t *TokenProcessorEvent) Message() interface{} {
	return t.message
}

func (cts *CommonTokenStore) Notify(topic string, walletID, tokenType, txID string, index uint64) {
	if cts.notifier == nil {
		logger.Warnf("cannot notify others!")
		return
	}

	e := NewTokenProcessorEvent(topic, &TokenMessage{
		WalletID:  walletID,
		TokenType: tokenType,
		TxID:      txID,
		Index:     index,
	})

	logger.Debugf("Publish new event %v", e)
	cts.notifier.Publish(e)
}
