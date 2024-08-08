/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

type mockConfigProvider struct{}

func (sp mockConfigProvider) UnmarshalKey(key string, rawVal interface{}) error { return nil }
func (sp mockConfigProvider) GetString(key string) string                       { return "" }
func (sp mockConfigProvider) GetBool(key string) bool                           { return false }
func (sp mockConfigProvider) IsSet(key string) bool                             { return false }
func (sp mockConfigProvider) TranslatePath(path string) string                  { return "" }

//func TestMemory(t *testing.T) {
//	d := NewDriver()
//
//	for _, c := range dbtest.Cases {
//		db, err := d.Driver.Open(new(mockConfigProvider), token.TMSID{Network: c.Name})
//		if err != nil {
//			t.Fatal(err)
//		}
//		t.Run(c.Name, func(xt *testing.T) {
//			defer db.Close()
//			c.Fn(xt, db)
//		})
//	}
//}
