/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

const Extension = `
token:
  tms: 
  - network: {{ TMS.Network }}
    channel: {{ TMS.Channel }}
    namespace: {{ TMS.Namespace }}
    certification: 
      interactive:
        ids: {{ range TMS.Certifiers }}
        - {{ . }}{{ end }}
    {{ if Wallets }}wallets:{{ if Wallets.Certifiers }}
      certifiers: {{ range Wallets.Certifiers }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}{{ if Wallets.Issuers }}
      issuers: {{ range Wallets.Issuers }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}{{ if Wallets.Owners }}
      owners: {{ range Wallets.Owners }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}{{ if Wallets.Auditors }}
      auditors: {{ range Wallets.Auditors }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}
    {{ end }}
`
