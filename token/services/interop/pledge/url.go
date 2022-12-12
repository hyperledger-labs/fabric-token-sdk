/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package pledge

import (
	"fmt"
	url2 "net/url"
	"strings"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/pkg/errors"
)

func FabricURL(tms token.TMSID) string {
	return fmt.Sprintf("fabric://%s.%s.%s/", tms.Network, tms.Channel, tms.Namespace)
}

func FabricURLToTMSID(url string) (token.TMSID, error) {
	u, err := url2.Parse(url)
	if err != nil {
		return token.TMSID{}, errors.Wrapf(err, "failed parsing url")
	}
	if u.Scheme != "fabric" {
		return token.TMSID{}, errors.Errorf("invalid scheme, expected fabric, got [%s]", u.Scheme)
	}

	res := strings.Split(u.Host, ".")
	if len(res) != 3 {
		return token.TMSID{}, errors.Errorf("invalid host, expected 3 components, found [%d,%v]", len(res), res)
	}
	return token.TMSID{
		Network: res[0], Channel: res[1], Namespace: res[2],
	}, nil
}
