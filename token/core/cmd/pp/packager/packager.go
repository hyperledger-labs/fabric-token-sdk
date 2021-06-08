/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package packager

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"path/filepath"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/persistence"
	"github.com/pkg/errors"
)

// Writer defines the interface needed for writing a file
type Writer interface {
	WriteFile(string, string, []byte) error
}

// PackageInput holds the input parameters for packaging a
// ChaincodeInstallPackage
type PackageInput struct {
	OutputFile string
	Path       string
	Type       string
	Label      string
}

// Validate checks for the required inputs
func (p *PackageInput) Validate() error {
	if p.Path == "" {
		return errors.New("chaincode path must be specified")
	}
	if p.Type == "" {
		return errors.New("chaincode language must be specified")
	}
	if p.OutputFile == "" {
		return errors.New("output file must be specified")
	}
	if p.Label == "" {
		return errors.New("package label must be specified")
	}
	if err := persistence.ValidateLabel(p.Label); err != nil {
		return err
	}

	return nil
}

// PlatformRegistry defines the interface to get the code bytes
// for a chaincode given the type and path
type PlatformRegistry interface {
	GetDeploymentPayload(ccType, path string, replacer func(string, string) []byte) ([]byte, error)
	NormalizePath(ccType, path string) (string, error)
}

// Packager holds the dependencies needed to package
// a chaincode and write it
type Packager struct {
	Input            *PackageInput
	PlatformRegistry PlatformRegistry
	Writer           Writer
}

func New() *Packager {
	pr := NewRegistry(SupportedPlatforms...)
	return &Packager{
		PlatformRegistry: pr,
		Writer:           &persistence.FilesystemIO{},
	}
}

// PackageChaincode packages a chaincode.
func (p *Packager) PackageChaincode(path, typ, label, outputFile string, replacer func(string, string) []byte) error {
	p.setInput(path, typ, label, outputFile)
	return p.Package(replacer)
}

func (p *Packager) setInput(path, typ, label, outputFile string) {
	p.Input = &PackageInput{
		OutputFile: outputFile,
		Path:       path,
		Type:       typ,
		Label:      label,
	}
}

// Package packages chaincodes into the package type,
// (.tar.gz) used by _lifecycle and writes it to disk
func (p *Packager) Package(replacer func(string, string) []byte) error {
	err := p.Input.Validate()
	if err != nil {
		return err
	}

	pkgTarGzBytes, err := p.getTarGzBytes(replacer)
	if err != nil {
		return err
	}

	dir, name := filepath.Split(p.Input.OutputFile)
	// if p.Input.OutputFile is only file name, dir becomes an empty string that creates problem
	// while invoking 'WriteFile' function below. So, irrespective, translate dir into absolute path
	if dir, err = filepath.Abs(dir); err != nil {
		return err
	}
	err = p.Writer.WriteFile(dir, name, pkgTarGzBytes)
	if err != nil {
		err = errors.Wrapf(err, "error writing chaincode package to %s", p.Input.OutputFile)
		logger.Error(err.Error())
		return err
	}

	return nil
}

func (p *Packager) getTarGzBytes(replacer func(string, string) []byte) ([]byte, error) {
	payload := bytes.NewBuffer(nil)
	gw := gzip.NewWriter(payload)
	tw := tar.NewWriter(gw)

	normalizedPath, err := p.PlatformRegistry.NormalizePath(strings.ToUpper(p.Input.Type), p.Input.Path)
	if err != nil {
		return nil, errors.WithMessage(err, "failed to normalize chaincode path")
	}
	metadataBytes, err := toJSON(normalizedPath, p.Input.Type, p.Input.Label)
	if err != nil {
		return nil, err
	}
	err = writeBytesToPackage(tw, "metadata.json", metadataBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error writing package metadata to tar")
	}

	codeBytes, err := p.PlatformRegistry.GetDeploymentPayload(strings.ToUpper(p.Input.Type), p.Input.Path, replacer)
	if err != nil {
		return nil, errors.WithMessage(err, "error getting chaincode bytes")
	}

	codePackageName := "code.tar.gz"

	err = writeBytesToPackage(tw, codePackageName, codeBytes)
	if err != nil {
		return nil, errors.Wrap(err, "error writing package code bytes to tar")
	}

	err = tw.Close()
	if err == nil {
		err = gw.Close()
	}
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create tar for chaincode")
	}

	return payload.Bytes(), nil
}

func writeBytesToPackage(tw *tar.Writer, name string, payload []byte) error {
	err := tw.WriteHeader(&tar.Header{
		Name: name,
		Size: int64(len(payload)),
		Mode: 0100644,
	})
	if err != nil {
		return err
	}

	_, err = tw.Write(payload)
	if err != nil {
		return err
	}

	return nil
}

// PackageMetadata holds the path and type for a chaincode package
type PackageMetadata struct {
	Path  string `json:"path"`
	Type  string `json:"type"`
	Label string `json:"label"`
}

func toJSON(path, ccType, label string) ([]byte, error) {
	metadata := &PackageMetadata{
		Path:  path,
		Type:  ccType,
		Label: label,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal chaincode package metadata into JSON")
	}

	return metadataBytes, nil
}
