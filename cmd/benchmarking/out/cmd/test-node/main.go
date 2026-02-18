/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	fscnode "github.com/hyperledger-labs/fabric-smart-client/node"
)

func main() {
	n := fscnode.New()
	n.Execute(func() error {
		return nil
	})
}
