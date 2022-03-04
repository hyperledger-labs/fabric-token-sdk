/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"fmt"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
)

type BackedTopology interface {
	Name() string
}

type Chaincode struct {
	Orgs        []string
	Private     bool
	DockerImage string
}

type TMS struct {
	Network             string
	Channel             string
	Namespace           string
	Driver              string
	PublicParamsGenArgs []string
	Auditors            []string
	Certifiers          []string
	Issuers             []string

	BackendTopology BackedTopology `yaml:"-"`
	BackendParams   map[string]interface{}
}

func (t *TMS) AddAuditor(auditor *node.Node) *TMS {
	t.Auditors = append(t.Auditors, auditor.Name)
	return t
}

func (t *TMS) AddCertifier(certifier *node.Node) *TMS {
	t.Certifiers = append(t.Certifiers, certifier.Name)
	return t
}

func (t *TMS) AddIssuer(issuer *node.Node) *TMS {
	t.Issuers = append(t.Issuers, issuer.Name)
	return t
}

func (t *TMS) SetTokenGenPublicParams(publicParamsGenArgs ...string) {
	t.PublicParamsGenArgs = publicParamsGenArgs
}

func (t *TMS) ID() string {
	return fmt.Sprintf("%s-%s-%s", t.Network, t.Channel, t.Network)
}
