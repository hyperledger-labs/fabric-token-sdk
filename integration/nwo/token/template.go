/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

const Extension = `
token:
  version: v1
  enabled: true
  selector:
    driver: {{ TokenSelector }}
  finality:
    # we leave type empty so that the default is peaked per type of network
    type:
    delivery:
      mapperParallelism: 10
      lruSize: 100
      lruBuffer: 50
      listenerTimeout: 30s
`
