/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"text/template"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common/runner"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
)

type CAFactory = func(string) (CA, error)

type CA interface {
	Start() error
}

type CAServer struct {
	NetworkPrefix string
	ConfigPath    string
}

func (c CAServer) SessionName() string {
	return c.NetworkPrefix + "-fabric-ca-server"
}

func (c CAServer) Args() []string {
	return []string{
		"start",
		"--config", c.ConfigPath,
	}
}

type IdemixCASupport struct {
	IssuerCryptoMaterialPath string
	ColorIndex               int
	StartEventuallyTimeout   time.Duration
}

func NewIdemixCASupport(issuerCryptoMaterialPath string) (CA, error) {
	return &IdemixCASupport{
		IssuerCryptoMaterialPath: issuerCryptoMaterialPath,
		StartEventuallyTimeout:   1 * time.Minute,
	}, nil
}

func (i *IdemixCASupport) Start() error {
	// generate configuration
	if err := i.GenerateConfiguration(); err != nil {
		return errors.Wrap(err, "failed to generate fabric-ca-server configuration")
	}

	// start
	fabricCAServerExePath := findCmdAtEnv(fabricCaServerCMD)
	command := &CAServer{
		NetworkPrefix: "",
		ConfigPath:    filepath.Join(i.IssuerCryptoMaterialPath, "ca", "fabric-ca-server.yaml"),
	}
	cmd := common.NewCommand(fabricCAServerExePath, command)

	config := runner.Config{
		AnsiColorCode:     i.nextColor(),
		Name:              command.SessionName(),
		Command:           cmd,
		StartCheck:        `Listening on .*`,
		StartCheckTimeout: 1 * time.Minute,
	}
	members := grouper.Members{}
	members = append(members, grouper.Member{Name: command.SessionName(), Runner: runner.New(config)})
	process := ifrit.Invoke(grouper.NewOrdered(syscall.SIGTERM, members))
	Eventually(process.Ready(), i.StartEventuallyTimeout).Should(BeClosed(), "fabric-ca-server didn't start timely")

	return nil
}

func (i *IdemixCASupport) GenerateConfiguration() error {
	t, err := template.New("fabric-ca-server").Funcs(template.FuncMap{
		"issuerpublickeyfile": func() string {
			return filepath.Join(i.IssuerCryptoMaterialPath, "ca", "IssuerPublicKey")
		},
		"issuersecretkeyfile": func() string {
			return filepath.Join(i.IssuerCryptoMaterialPath, "ca", "IssuerSecretKey")
		},
		"revocationpublickeyfile": func() string {
			return filepath.Join(i.IssuerCryptoMaterialPath, "msp", "RevocationPublicKey")
		},
		"revocationprivatekeyfile": func() string {
			return filepath.Join(i.IssuerCryptoMaterialPath, "ca", "RevocationKey")
		},
	}).Parse(CACfgTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to prepare fabric-ca-server configuration template")
	}
	ext := bytes.NewBufferString("")
	if err := t.Execute(io.MultiWriter(ext), i); err != nil {
		return errors.Wrap(err, "failed to generate fabric-ca-server configuration")
	}
	if err := os.WriteFile(filepath.Join(i.IssuerCryptoMaterialPath, "ca", "fabric-ca-server.yaml"), ext.Bytes(), 0766); err != nil {
		return errors.Wrap(err, "failed to write fabric-ca-server configuration")
	}
	return nil
}

func (i *IdemixCASupport) nextColor() string {
	color := i.ColorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	i.ColorIndex++
	return fmt.Sprintf("%dm", color)
}
