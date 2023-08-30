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
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common"
	"github.com/hyperledger-labs/fabric-smart-client/integration/nwo/common/runner"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/generators"
	"github.com/hyperledger-labs/fabric-token-sdk/integration/nwo/token/topology"
	"github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/pkg/errors"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
)

type CAFactory = func(generators.TokenPlatform, *topology.TMS, string) (CA, error)

type CA interface {
	Start() error
	Stop()
	Gen(owner string) (string, error)
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
		"-d",
		"--config", c.ConfigPath,
	}
}

type CAClientRegister struct {
	NetworkPrefix  string
	CAServerURL    string
	CAName         string
	IDName         string
	IDSecret       string
	IDType         string
	EnrollmentType string
	IdemixCurve    string
	MSPDir         string
}

func (c CAClientRegister) SessionName() string {
	return c.NetworkPrefix + "-fabric-ca-client-register"
}

func (c CAClientRegister) Args() []string {
	return []string{
		"register",
		"-d",
		"--mspdir", c.MSPDir,
		"-u", c.CAServerURL,
		"--caname", c.CAName,
		"--id.name", c.IDName,
		"--id.secret", c.IDSecret,
		"--id.type", c.IDType,
		"--enrollment.type", c.EnrollmentType,
		"--idemix.curve", c.IdemixCurve,
	}
}

type CAClientEnroll struct {
	NetworkPrefix  string
	Home           string
	CAServerURL    string
	CAName         string
	Output         string
	EnrollmentType string
	IdemixCurve    string
}

func (c CAClientEnroll) SessionName() string {
	return c.NetworkPrefix + "-fabric-ca-client-enroll"
}

func (c CAClientEnroll) Args() []string {
	return []string{
		"enroll",
		"-d",
		"--home", c.Home,
		"-u", c.CAServerURL,
		"--caname", c.CAName,
		"-M", c.Output,
		"--enrollment.type", c.EnrollmentType,
		"--idemix.curve", c.IdemixCurve,
	}
}

type IdemixCASupport struct {
	IssuerCryptoMaterialPath string
	ColorIndex               int
	StartEventuallyTimeout   time.Duration
	EventuallyTimeout        time.Duration
	process                  ifrit.Process
	TokenPlatform            generators.TokenPlatform
	TMS                      *topology.TMS
	CAPort                   string
}

func NewIdemixCASupport(tokenPlatform generators.TokenPlatform, tms *topology.TMS, issuerCryptoMaterialPath string) (CA, error) {
	return &IdemixCASupport{
		IssuerCryptoMaterialPath: issuerCryptoMaterialPath,
		StartEventuallyTimeout:   1 * time.Minute,
		EventuallyTimeout:        1 * time.Minute,
		TokenPlatform:            tokenPlatform,
		TMS:                      tms,
		CAPort:                   "7054",
	}, nil
}

func (i *IdemixCASupport) Start() error {
	// generate configuration
	if err := i.GenerateConfiguration(); err != nil {
		return errors.Wrap(err, "failed to generate fabric-ca-server configuration")
	}

	// start fabric-ca-server
	fabricCAServerExePath := findCmdAtEnv(fabricCaServerCMD)
	command := &CAServer{
		NetworkPrefix: "",
		ConfigPath:    filepath.Join(i.IssuerCryptoMaterialPath, "fabric-ca-server", "fabric-ca-server.yaml"),
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
	i.process = ifrit.Invoke(grouper.NewOrdered(syscall.SIGTERM, members))
	Eventually(i.process.Ready(), i.StartEventuallyTimeout).Should(BeClosed(), "fabric-ca-server didn't start timely")

	// enroll admin
	caName := i.TMS.ID() + ".example.com"
	fabricCAClientExePath := findCmdAtEnv(fabricCaClientCMD)
	enrollCommand := &CAClientEnroll{
		Home:        "",
		CAServerURL: fmt.Sprintf("http://%s:%s@localhost:%s", "admin", "adminpw", i.CAPort),
		CAName:      caName,
		Output:      filepath.Join(i.IssuerCryptoMaterialPath, "fabric-ca-server", "admin", "msp"),
	}
	cmd = common.NewCommand(fabricCAClientExePath, enrollCommand)
	sess, err := i.StartSession(cmd, enrollCommand.SessionName())
	if err != nil {
		return errors.Wrap(err, "failed to start admin enrollment")
	}
	Eventually(sess, i.EventuallyTimeout).Should(gexec.Exit(0))

	return nil
}

func (i *IdemixCASupport) Stop() {
	if i.process != nil {
		i.process.Signal(syscall.SIGTERM)
	}
}

func (i *IdemixCASupport) Gen(owner string) (string, error) {
	//fabric-ca-client register --caname ca-org1 --id.name owner1a --id.secret password --id.type client --enrollment.type idemix --idemix.curve gurvy.Bn254 --tls.certfiles "${CERT_FILES}"
	//fabric-ca-client enroll -u https://owner1a:password@localhost:7054 --caname ca-org1  -M "${PWD}/keys/owner1/wallet/owner1a/msp" --enrollment.type idemix --idemix.curve gurvy.Bn254 --tls.certfiles "${CERT_FILES}"

	tmsID := i.TMS.ID()
	logger.Debugf("Generating owner identity [%s] for [%s]", owner, tmsID)
	userOutput := filepath.Join(i.TokenPlatform.TokenDir(), "crypto", tmsID, "idemix", owner)
	if err := os.MkdirAll(userOutput, 0766); err != nil {
		return "", nil
	}

	caName := tmsID + ".example.com"
	fabricCAClientExePath := findCmdAtEnv(fabricCaClientCMD)

	// register
	registerCommand := &CAClientRegister{
		MSPDir:         filepath.Join(i.IssuerCryptoMaterialPath, "fabric-ca-server", "admin", "msp"),
		CAServerURL:    fmt.Sprintf("http://localhost:%s", i.CAPort),
		CAName:         caName,
		IDName:         owner,
		IDSecret:       "password",
		IDType:         "client",
		EnrollmentType: "idemix",
		IdemixCurve:    "gurvy.Bn254",
	}
	cmd := common.NewCommand(fabricCAClientExePath, registerCommand)
	sess, err := i.StartSession(cmd, registerCommand.SessionName())
	if err != nil {
		return "", err
	}
	Eventually(sess, i.EventuallyTimeout).Should(gexec.Exit(0))

	enrollCommand := &CAClientEnroll{
		Home:           "",
		CAServerURL:    fmt.Sprintf("http://%s:%s@localhost:%s", registerCommand.IDName, registerCommand.IDSecret, i.CAPort),
		CAName:         caName,
		Output:         userOutput,
		EnrollmentType: "idemix",
		IdemixCurve:    "gurvy.Bn254",
	}
	cmd = common.NewCommand(fabricCAClientExePath, enrollCommand)
	sess, err = i.StartSession(cmd, enrollCommand.SessionName())
	if err != nil {
		return "", err
	}
	Eventually(sess, i.EventuallyTimeout).Should(gexec.Exit(0))

	return userOutput, nil
}

func (i *IdemixCASupport) GenerateConfiguration() error {
	fabricCARoot := filepath.Join(i.IssuerCryptoMaterialPath, "fabric-ca-server")
	if err := os.MkdirAll(fabricCARoot, 0766); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(fabricCARoot, "msp", "keystore"), 0766); err != nil {
		return err
	}
	if err := CopyFile(filepath.Join(i.IssuerCryptoMaterialPath, "ca", "IssuerPublicKey"), filepath.Join(fabricCARoot, "IssuerPublicKey")); err != nil {
		return err
	}
	if err := CopyFile(filepath.Join(i.IssuerCryptoMaterialPath, "msp", "RevocationPublicKey"), filepath.Join(fabricCARoot, "IssuerRevocationPublicKey")); err != nil {
		return err
	}
	if err := CopyFile(filepath.Join(i.IssuerCryptoMaterialPath, "ca", "IssuerSecretKey"), filepath.Join(fabricCARoot, "msp", "keystore", "IssuerSecretKey")); err != nil {
		return err
	}
	if err := CopyFile(filepath.Join(i.IssuerCryptoMaterialPath, "ca", "RevocationKey"), filepath.Join(fabricCARoot, "msp", "keystore", "IssuerRevocationPrivateKey")); err != nil {
		return err
	}

	t, err := template.New("fabric-ca-server").Funcs(template.FuncMap{
		"caname": func() string {
			return i.TMS.ID() + ".example.com"
		},
		"Port": func() string {
			i.CAPort = fmt.Sprintf("%d", i.TokenPlatform.GetContext().ReservePort())
			return i.CAPort
		},
	}).Parse(CACfgTemplate)
	if err != nil {
		return errors.Wrap(err, "failed to prepare fabric-ca-server configuration template")
	}
	ext := bytes.NewBufferString("")
	if err := t.Execute(io.MultiWriter(ext), i); err != nil {
		return errors.Wrap(err, "failed to generate fabric-ca-server configuration")
	}
	if err := os.MkdirAll(filepath.Join(i.IssuerCryptoMaterialPath, "fabric-ca-server"), 0766); err != nil {
		return errors.Wrap(err, "failed to create fabric-ca-server configuration folder")
	}
	if err := os.WriteFile(filepath.Join(i.IssuerCryptoMaterialPath, "fabric-ca-server", "fabric-ca-server.yaml"), ext.Bytes(), 0766); err != nil {
		return errors.Wrap(err, "failed to write fabric-ca-server configuration")
	}
	return nil
}

func (i *IdemixCASupport) StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
	ansiColorCode := i.nextColor()
	fmt.Fprintf(
		ginkgo.GinkgoWriter,
		"\x1b[33m[d]\x1b[%s[%s]\x1b[0m starting %s %s\n",
		ansiColorCode,
		name,
		filepath.Base(cmd.Args[0]),
		strings.Join(cmd.Args[1:], " "),
	)
	return gexec.Start(
		cmd,
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
	)
}

func (i *IdemixCASupport) nextColor() string {
	color := i.ColorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	i.ColorIndex++
	return fmt.Sprintf("%dm", color)
}

func CopyFile(src, dst string) error {
	cleanSrc := filepath.Clean(src)
	cleanDst := filepath.Clean(dst)
	if cleanSrc == cleanDst {
		return nil
	}
	sf, err := os.Open(cleanSrc)
	if err != nil {
		return err
	}
	defer sf.Close()
	df, err := os.Create(cleanDst)
	if err != nil {
		return err
	}
	defer df.Close()
	_, err = io.Copy(df, sf)
	return err
}
