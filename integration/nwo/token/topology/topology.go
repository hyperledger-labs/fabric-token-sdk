/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import (
	"fmt"
	"regexp"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
)

var (
	nsRegexp = regexp.MustCompile("^[a-zA-Z0-9._-]{1,120}$")
)

// ValidateNs checks if the namespace is valid
func ValidateNs(ns string) error {
	if !nsRegexp.MatchString(ns) {
		return errors.Errorf("namespace '%s' is invalid", ns)
	}

	return nil
}

type BackedTopology interface {
	Name() string
}

// TokenTopology models the topology of the token network
type TokenTopology interface {
	// GetTMSs returns the list of TMSs in the topology
	GetTMSs() []*TMS
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

	TokenTopology   TokenTopology  `yaml:"-"`
	FSCNodes        []*node.Node   `yaml:"-"`
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
	return fmt.Sprintf("%s-%s-%s", t.Network, t.Channel, t.Namespace)
}

// SetNamespace sets the namespace of the TMS
func (t *TMS) SetNamespace(namespace string) *TMS {
	// Check if namespace is valid
	err := ValidateNs(namespace)
	if err != nil {
		panic(errors.WithMessagef(err, "invalid namespace '%s'", namespace))
	}
	// Check if another namespace already exists in the same <network,channel>
	for _, tms := range t.TokenTopology.GetTMSs() {
		if tms.Network == t.Network && tms.Channel == t.Channel {
			Expect(tms.Namespace).To(Not(Equal(namespace)), "Namespace [%s] already exists", namespace)
		}
	}

	t.Namespace = namespace
	return t
}
