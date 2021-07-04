/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

const Extension = `
token:
  enabled: true
  auditor:
    auditdb:
      persistence:
        type: badger
        opts:
          path: {{ NodeKVSPath }}
  tms: {{ range TMSs }}
  - network: {{ .Network }}
    channel: {{ .Channel }}
    namespace: {{ .Namespace }}
    certification: 
      interactive:
        ids: {{ range .Certifiers }}
        - {{ . }}{{ end }}
    {{ if Wallets }}wallets:{{ if Wallets.Certifiers }}
      certifiers: {{ range Wallets.Certifiers }}
      - id: {{ .ID }}
        mspType: {{ .MSPType }}
        mspID: {{ .MSPID }}
        path: {{ .Path }}
      {{ end }}
      {{ end }}
    {{ end }}
  {{ end }}
`
