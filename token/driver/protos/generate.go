/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protos

//go:generate protoc request.proto --go_out=../protos-go/request --go_opt=paths=source_relative
//go:generate protoc pp.proto --go_out=../protos-go/pp --go_opt=paths=source_relative
