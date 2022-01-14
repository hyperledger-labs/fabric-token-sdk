/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package topology

import "github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fsc/node"

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

func (o *Options) Issuers() []string {
	boxed := o.Mapping["Issuers"]
	if boxed == nil {
		return nil
	}
	res, ok := boxed.([]string)
	if ok {
		return res
	}
	res = []string{}
	for _, v := range boxed.([]interface{}) {
		res = append(res, v.(string))
	}
	return res
}

func (o *Options) SetIssuers(ids []string) {
	o.Mapping["Issuers"] = ids
}

func (o *Options) Owners() []string {
	boxed := o.Mapping["Owners"]
	if boxed == nil {
		return nil
	}
	res, ok := boxed.([]string)
	if ok {
		return res
	}
	res = []string{}
	for _, v := range boxed.([]interface{}) {
		res = append(res, v.(string))
	}
	return res
}

func (o *Options) SetOwners(ids []string) {
	o.Mapping["Owners"] = ids
}

func (o *Options) Auditor() bool {
	res := o.Mapping["Auditor"]
	if res == nil {
		return false
	}
	return res.(bool)
}

func (o *Options) SetAuditor(v bool) {
	o.Mapping["Auditor"] = v
}

func ToOptions(o *node.Options) *Options {
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
		return Convert(mapping)
	}
	panic("invalid options")
}

func Convert(m map[interface{}]interface{}) *Options {
	opts := &Options{
		Mapping: map[string]interface{}{},
	}
	for k, v := range m["mapping"].(map[interface{}]interface{}) {
		opts.Mapping[k.(string)] = v
	}
	return opts
}
