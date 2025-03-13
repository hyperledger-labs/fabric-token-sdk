/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabricv25

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/fabric/packager"
	"github.com/pkg/errors"
)

// GeneratePackage generates the chaincode package for the given raw public parameters
func GeneratePackage(raw []byte, outputDir string) error {
	t, err := template.New("node").Funcs(template.FuncMap{
		"Params": func() string { return base64.StdEncoding.EncodeToString(raw) },
	}).Parse(DefaultParams)
	if err != nil {
		return errors.Wrap(err, "failed creating params template")
	}
	paramsFile := bytes.NewBuffer(nil)
	err = t.Execute(io.MultiWriter(paramsFile), nil)
	if err != nil {
		return errors.Wrap(err, "failed writing params template")
	}

	err = packager.New().PackageChaincode(
		"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/tcc/fabricv25/main",
		"golang",
		"tcc",
		filepath.Join(outputDir, "tcc.tar"),
		func(s string, s2 string) (string, []byte) {
			fmt.Println(s)
			if strings.HasSuffix(s, "github.com/hyperledger-labs/fabric-token-sdk/token/tcc/fabricv25/params.go") {
				return "", paramsFile.Bytes()
			}
			return "", nil
		},
	)
	if err != nil {
		return errors.Wrap(err, "failed creating chaincode package")
	}

	return nil
}
