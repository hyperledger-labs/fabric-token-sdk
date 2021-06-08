/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package external

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/pkg/errors"
)

// Directory constant copied from tar package.
const c_ISDIR = 040000

// Default compression to use for production. Test packages disable compression.
var gzipCompressionLevel = gzip.DefaultCompression

// Platform for external chaincodes
type Platform struct{}

// Name returns the name of this platform.
func (p *Platform) Name() string {
	return "EXTERNAL"
}

// ValidatePath is used to ensure that path provided points to something that
// looks like go chainccode.
//
// NOTE: this is only used at the _client_ side by the peer CLI.
func (p *Platform) ValidatePath(rawPath string) error {
	return nil
}

// NormalizePath is used to extract a relative module path from a module root.
// This should not impact legacy GOPATH chaincode.
//
// NOTE: this is only used at the _client_ side by the peer CLI.
func (p *Platform) NormalizePath(rawPath string) (string, error) {
	return rawPath, nil
}

// ValidateCodePackage examines the chaincode archive to ensure it is valid.
//
// NOTE: this code is used in some transaction validation paths but can be changed
// post 2.0.
func (p *Platform) ValidateCodePackage(code []byte) error {
	is := bytes.NewReader(code)
	gr, err := gzip.NewReader(is)
	if err != nil {
		return fmt.Errorf("failure opening codepackage gzip stream: %s", err)
	}

	re := regexp.MustCompile(`^(src|META-INF)/`)
	tr := tar.NewReader(gr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// maintain check for conforming paths for validation
		if !re.MatchString(header.Name) {
			return fmt.Errorf("illegal file name in payload: %s", header.Name)
		}

		// only files and directories; no links or special files
		mode := header.FileInfo().Mode()
		if mode&^(os.ModeDir|0777) != 0 {
			return fmt.Errorf("illegal file mode in payload: %s", header.Name)
		}
	}

	return nil
}

// GetDeploymentPayload creates a gzip compressed tape archive that contains the
// required assets to build and run go chaincode.
//
// NOTE: this is only used at the _client_ side by the peer CLI.
func (p *Platform) GetDeploymentPayload(codepath string, replacer func(string, string) []byte) ([]byte, error) {
	payload := bytes.NewBuffer(nil)
	gw, err := gzip.NewWriterLevel(payload, gzipCompressionLevel)
	if err != nil {
		return nil, err
	}
	tw := tar.NewWriter(gw)

	raw := replacer("connection.json", "connection.json")
	if err := WriteBytesToPackage(raw, "connection.json", tw); err != nil {
		return nil, fmt.Errorf("error writing connection.json to tar: %s", err)
	}
	if err != nil {
		return nil, fmt.Errorf("error writing connection.json to tar: %s", err)
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

// WriteBytesToPackage writes a file to a tar stream.
func WriteBytesToPackage(raw []byte, packagepath string, tw *tar.Writer) error {
	// Take the variance out of the tar by using zero time and fixed uid/gid.
	var zeroTime time.Time
	header := &tar.Header{}
	header.AccessTime = zeroTime
	header.ModTime = zeroTime
	header.ChangeTime = zeroTime
	header.Name = packagepath
	header.Mode = 0100644
	header.Uid = 500
	header.Gid = 500
	header.Uname = ""
	header.Gname = ""
	header.Size = int64(len(raw))

	err := tw.WriteHeader(header)
	if err != nil {
		return fmt.Errorf("failed to write header for %s", err)
	}

	_, err = io.Copy(tw, bytes.NewBuffer(raw))
	if err != nil {
		return fmt.Errorf("failed to write as %s: %s", packagepath, err)
	}

	return nil
}
