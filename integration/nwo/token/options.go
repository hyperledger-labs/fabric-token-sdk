/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package token

import (
	"fmt"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric"
	fsc "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"
)

type Options struct {
	Mapping map[string]interface{}
}

func (o *Options) Certifier() bool {
	res := o.Mapping["Certifier"]
	if res == nil {
		return false
	}
	return res.(bool)
}

func (o *Options) SetCertifier(v bool) {
	o.Mapping["Certifier"] = v
}

func options(o *fsc.Options) *Options {
	opt, ok := o.Mapping["token"]
	if !ok {
		opt = &Options{Mapping: map[string]interface{}{}}
		o.Mapping["token"] = opt
	}
	res, ok := opt.(*Options)
	if ok {
		return res
	}
	mapping, ok := opt.(map[interface{}]interface{})
	if ok {
		return convert(mapping)
	}
	panic("invalid options")
}

func convert(m map[interface{}]interface{}) *Options {
	opts := &Options{
		Mapping: map[string]interface{}{},
	}
	for k, v := range m["mapping"].(map[interface{}]interface{}) {
		opts.Mapping[k.(string)] = v
	}
	return opts
}

func WithIssuerIdentity(label string) fsc.Option {
	return func(o *fsc.Options) error {
		fo := fabric.Options(o)
		fo.SetX509Identities(append(fo.X509Identities(), label))
		return nil
	}
}

func WithOwnerIdentity(driver string, label string) fsc.Option {
	return func(o *fsc.Options) error {
		fo := fabric.Options(o)
		switch driver {
		case "dlog":
			fo.SetIdemixIdentities(append(fo.IdemixIdentities(), label))
		case "fabtoken":
			fo.SetX509Identities(append(fo.X509Identities(), label))
		default:
			panic(fmt.Sprintf("unexpected driver [%s]", driver))
		}
		return nil
	}
}

func WithCertifierIdentity() fsc.Option {
	return func(o *fsc.Options) error {
		options(o).SetCertifier(true)

		return nil
	}
}
