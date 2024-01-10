/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	url2 "net/url"

	"github.com/pkg/errors"
)

// CheckFabricScheme returns an error is the given url is not valid or its scheme is not equal to fabric
func CheckFabricScheme(url string) error {
	u, err := url2.Parse(url)
	if err != nil {
		return errors.Wrapf(err, "failed parsing url [%s]", url)
	}
	if u.Scheme != "fabric" {
		return errors.Errorf("invalid scheme, expected fabric, got [%s] in url [%s]", u.Scheme, url)
	}
	return nil
}
