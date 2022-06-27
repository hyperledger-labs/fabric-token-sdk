/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

const Extension = `
token:
  enabled: true
  ttxdb:
    persistence:
      type: badger
      opts:
        path: {{ NodeKVSPath }}
`
