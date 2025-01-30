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
    type: {{ FinalityType }}
    delivery:
      mapperParallelism: 10
      lruSize: 100
      lruBuffer: 50
      listenerTimeout: 30s
`
