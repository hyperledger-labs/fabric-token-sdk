/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package identity

import (
	"fmt"

	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	view2 "github.com/hyperledger-labs/fabric-smart-client/platform/view"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/hash"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/api"
)

var logger = flogging.MustGetLogger("token-sdk.driver.identity.fabric")

type GetFunc func() (view.Identity, []byte, error)

type Mapper interface {
	Info(id string) (string, string, GetFunc)
	Map(v interface{}) (view.Identity, string)
}

type Provider struct {
	sp view2.ServiceProvider

	mappers map[api.IdentityUsage]Mapper
}

func NewProvider(sp view2.ServiceProvider, mappers map[api.IdentityUsage]Mapper) *Provider {
	return &Provider{
		sp:      sp,
		mappers: mappers,
	}
}

func (i *Provider) GetIdentityInfo(usage api.IdentityUsage, id string) *api.IdentityInfo {
	mapper, ok := i.mappers[usage]
	if !ok {
		panic(fmt.Sprintf("mapper not found for usage [%d]", usage))
	}
	id, eid, getIdentity := mapper.Info(id)
	if getIdentity == nil {
		return nil
	}
	logger.Debugf("info for [%v] is [%s,%s]", id, id, eid)
	return &api.IdentityInfo{
		ID:           id,
		EnrollmentID: eid,
		GetIdentity: func() (view.Identity, error) {
			id, ai, err := getIdentity()
			if err != nil {
				return nil, err
			}
			if err := i.RegisterAuditInfo(id, ai); err != nil {
				return nil, err
			}
			if err := view2.GetEndpointService(i.sp).Bind(view2.GetIdentityProvider(i.sp).DefaultIdentity(), id); err != nil {
				return nil, err
			}
			return id, nil
		},
	}
}

func (i *Provider) LookupIdentifier(usage api.IdentityUsage, v interface{}) (view.Identity, string) {
	mapper, ok := i.mappers[usage]
	if !ok {
		panic(fmt.Sprintf("mapper not found for usage [%d]", usage))
	}
	id, label := mapper.Map(v)
	logger.Debugf("identifier for [%v] is [%s,%s]", v, id, label)
	return id, label
}

func (i *Provider) GetAuditInfo(identity view.Identity) ([]byte, error) {
	auditInfo, err := view2.GetSigService(i.sp).GetAuditInfo(identity)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting audit info for recipient identity [%s]", identity.String())
	}
	return auditInfo, nil
}

func (i *Provider) GetIdentityMetadata(identity view.Identity) ([]byte, error) {
	panic("implement me")
}

func (i *Provider) GetSigner(identity view.Identity) (api.Signer, error) {
	signer, err := view2.GetSigService(i.sp).GetSigner(identity)
	if err != nil {
		return nil, err
	}
	return signer, err
}

func (i *Provider) RegisterRecipientIdentity(id view.Identity, auditInfo, metadata []byte) error {
	logger.Debugf("register recipient identity [%s] with audit info [%s]", id.String(), hash.Hashable(auditInfo).String())

	// recognize identity and register it
	_, err := view2.GetSigService(i.sp).GetVerifier(id)
	if err != nil {
		return err
	}

	if err := view2.GetSigService(i.sp).RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}

	// TODO: register metadata

	return nil
}

func (i *Provider) RegisterAuditInfo(id view.Identity, auditInfo []byte) error {
	if err := view2.GetSigService(i.sp).RegisterAuditInfo(id, auditInfo); err != nil {
		return err
	}
	return nil
}

func (i *Provider) GetEnrollmentID(auditInfo []byte) (string, error) {
	ai := &idemix2.AuditInfo{}
	if err := ai.FromBytes(auditInfo); err != nil {
		return "", errors.Wrapf(err, "failed unamrshalling audit info [%s]", auditInfo)
	}
	return ai.EnrollmentID(), nil
}
