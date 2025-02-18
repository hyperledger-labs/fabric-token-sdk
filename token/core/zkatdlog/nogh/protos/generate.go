/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package protos

//go:generate protoc noghmath.proto --go_out=../protos-go/math --go_opt=paths=source_relative
//go:generate protoc  -I=. -I=../../../fabtoken/protos noghactions.proto --go_out=../protos-go/actions --go_opt=paths=source_relative
//go:generate protoc noghpp.proto --go_out=../protos-go/pp --go_opt=paths=source_relative
