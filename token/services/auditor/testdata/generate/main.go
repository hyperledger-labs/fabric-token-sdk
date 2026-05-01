/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/proto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/protos-go/request"
)

// main is the entry point for the fixture generator
func main() {
	baseDir := "token/services/auditor/testdata/regression"
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create base directory: %v\n", err)
		os.Exit(1)
	}

	generateV1(baseDir)
	generateV2(baseDir)
	generateV3(baseDir)
}

// generateV1 produces fixtures for Protocol V1
func generateV1(baseDir string) {
	fmt.Println("Generating Protocol V1 fixtures...")
	tr := &driver.TokenRequest{
		Version: driver.ProtocolV1,
		Issues:  [][]byte{[]byte("issue-v1")},
	}
	trm := &driver.TokenRequestMetadata{
		Issues: []*driver.IssueMetadata{
			{
				Issuer: driver.AuditableIdentity{Identity: driver.Identity("issuer-v1")},
				Outputs: []*driver.IssueOutputMetadata{
					{
						OutputMetadata: []byte("output-v1"),
						Receivers: []*driver.AuditableIdentity{
							{Identity: driver.Identity("receiver-v1")},
						},
					},
				},
			},
		},
	}
	save(baseDir, "v1", tr, trm)
}

// generateV2 produces fixtures for Protocol V2
func generateV2(baseDir string) {
	fmt.Println("Generating Protocol V2 fixtures...")
	tr := &driver.TokenRequest{
		Version: driver.ProtocolV2,
		Issues:  [][]byte{[]byte("issue-v2")},
	}
	trm := &driver.TokenRequestMetadata{
		Issues: []*driver.IssueMetadata{
			{
				Issuer: driver.AuditableIdentity{Identity: driver.Identity("issuer-v2")},
				Outputs: []*driver.IssueOutputMetadata{
					{
						OutputMetadata: []byte("output-v2"),
						Receivers: []*driver.AuditableIdentity{
							{Identity: driver.Identity("receiver-v2")},
						},
					},
				},
			},
		},
	}
	save(baseDir, "v2", tr, trm)
}

// generateV3 produces fixtures for Protocol V3
func generateV3(baseDir string) {
	fmt.Println("Generating Protocol V3 fixtures...")
	tr := &driver.TokenRequest{
		Version: driver.ProtocolV2, // TokenRequest still V2
		Issues:  [][]byte{[]byte("issue-v3")},
	}
	trm := &driver.TokenRequestMetadata{
		Issues: []*driver.IssueMetadata{
			{
				Issuer: driver.AuditableIdentity{Identity: driver.Identity("issuer-v3"), AuditInfo: []byte("issuer-audit-v3")},
				ExtraSigners: []*driver.AuditableIdentity{
					{Identity: driver.Identity("extra-v3"), AuditInfo: []byte("extra-audit-v3")},
				},
			},
		},
	}
	save(baseDir, "v3", tr, trm)
}

// save serializes the token request and metadata to a binary file
func save(baseDir, version string, tr *driver.TokenRequest, trm *driver.TokenRequestMetadata) {
	if _, err := tr.Bytes(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to serialize token request: %v\n", err)
		os.Exit(1)
	}
	if _, err := trm.Bytes(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to serialize token request metadata: %v\n", err)
		os.Exit(1)
	}

	// We wrap them in TokenRequestWithMetadata for easier loading
	trwm := &request.TokenRequestWithMetadata{
		Version:  driver.ProtocolV1, // This version is for the wrapper
		Anchor:   "anchor-" + version,
		Request:  toProtoRequest(tr),
		Metadata: toProtoMetadata(trm, version),
	}
	raw, err := proto.Marshal(trwm)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to marshal TokenRequestWithMetadata: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(filepath.Join(baseDir, version+".bin"), raw, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write fixture file: %v\n", err)
		os.Exit(1)
	}
}

// toProtoRequest converts a driver.TokenRequest to its protobuf representation
func toProtoRequest(tr *driver.TokenRequest) *request.TokenRequest {
	p, err := tr.ToProtos()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to convert TokenRequest to protos: %v\n", err)
		os.Exit(1)
	}
	return p
}

// toProtoMetadata converts a driver.TokenRequestMetadata to its protobuf representation
func toProtoMetadata(trm *driver.TokenRequestMetadata, version string) *request.TokenRequestMetadata {
	if version == "v3" {
		p, err := trm.ToProtos()
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to convert TokenRequestMetadata to protos: %v\n", err)
			os.Exit(1)
		}
		p.Version = driver.ProtocolV3
		return p
	}
	// For V1/V2, we manually create it to ensure it uses old fields if needed,
	// but trm.ToProtos() currently always uses V3.
	// So for V1/V2 we should use the old marshaling logic.

	p := &request.TokenRequestMetadata{
		Version: driver.ProtocolV2,
	}
	if version == "v1" {
		p.Version = driver.ProtocolV1
	}

	for _, issue := range trm.Issues {
		issueProto := &request.IssueMetadata{
			Issuer: &request.AuditableIdentity{
				Identity: &request.Identity{Raw: issue.Issuer.Identity},
			},
		}
		for _, out := range issue.Outputs {
			outProto := &request.OutputMetadata{
				Metadata: out.OutputMetadata,
			}
			for _, rec := range out.Receivers {
				outProto.Receivers = append(outProto.Receivers, &request.AuditableIdentity{
					Identity: &request.Identity{Raw: rec.Identity},
				})
			}
			issueProto.Outputs = append(issueProto.Outputs, outProto)
		}
		p.Metadata = append(p.Metadata, &request.ActionMetadata{
			Metadata: &request.ActionMetadata_IssueMetadata{
				IssueMetadata: issueProto,
			},
		})
	}
	return p
}
