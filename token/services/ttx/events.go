/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/events"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
)

func publisher(sp view2.ServiceProvider) (events.Publisher, error) {
	return events.GetPublisher(sp)
}

func publishAbortTx(pub events.Publisher, tx *Transaction) {

	logger.Debugf("processing abort")

	inputs, err := tx.Inputs()
	if err != nil {
		logger.Errorf("there is some error: ", err)
		return
	}

	for i := 0; i < inputs.Count(); i++ {
		input := inputs.At(i)
		e := processor.NewTokenProcessorEvent(processor.UpdateToken, &processor.TokenMessage{
			TxID:  input.Id.TxId,
			Index: input.Id.Index,
		})

		logger.Debugf("Publish on abort %v", e)
		pub.Publish(e)
	}

}
