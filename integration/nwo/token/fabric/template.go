/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

const Extension = `
token:
  # TMS stands for Token Management Service. A TMS is uniquely identified by a network, channel, and 
  # namespace identifiers. The network identifier should refer to a configure network (Fabric, Orion, and so on).
  # The meaning of channel and namespace are network dependant. For Fabric, the meaning is clear.
  # For Orion, channel is empty and namespace is the DB name to use.
  tms: 
  - # Network identifier this TMS refers to
    network: {{ TMS.Network }}
    # Channel identifier within the specified network
    channel: {{ TMS.Channel }}
	# Namespace identifier within the specified channel
    namespace: {{ TMS.Namespace }}
    certification: 
      interactive:
        ids: {{ range TMS.Certifiers }}
        - {{ . }}{{ end }}
    {{ if Wallets }} # Wallets associated with this TMS
    wallets:{{ if Wallets.Certifiers }}
      # Certifiers wallets are used to certify tokens
      certifiers: {{ range Wallets.Certifiers }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}{{ if Wallets.Issuers }}
	  # Issuers wallets are used to issue tokens
      issuers: {{ range Wallets.Issuers }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}{{ if Wallets.Owners }}
	  # Owners wallets are used to own tokens
      owners: {{ range Wallets.Owners }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}{{ if Wallets.Auditors }}
	  # Auditors wallets are used to audit tokens
      auditors: {{ range Wallets.Auditors }}
      - id: {{ .ID }}
        default: {{ .Default }}
        type: {{ .Type }}
        path: {{ .Path }}
      {{ end }}
    {{ end }}
    {{ end }}
`
