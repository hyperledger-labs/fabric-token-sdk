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
    {{ if SQL }}
      type: sql
      opts:
        createSchema: true 
        tablePrefix: tsdk  
        driver: sqlite    
        maxOpenConns: 10
        dataSource: {{ SQLDataSource }}
    {{ else }}
      type: badger
      opts:
        path: {{ NodeKVSPath }}
    {{ end }}
`
