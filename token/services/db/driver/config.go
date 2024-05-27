/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

type ConfigProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	GetString(key string) string
	IsSet(key string) bool
	TranslatePath(path string) string
}
