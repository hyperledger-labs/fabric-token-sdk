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
    type: delivery
    delivery:
      mapperParallelism: 10
      lruSize: 1500
      lruBuffer: 500
      listenerTimeout: 30s
`
