package kvs

import (
	"time"

	memory "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
)

func NewInMemory() (KVS, error) {
	configService := &fakeProv{typ: "memory"}
	return kvs.NewWithConfig(&memory.Driver{}, "", configService)
}

type fakeProv struct {
	typ string
}

func (f *fakeProv) GetString(key string) string {
	return f.typ
}

func (f *fakeProv) GetInt(key string) int {
	return 0
}

func (f *fakeProv) GetDuration(key string) time.Duration {
	return time.Duration(0)
}

func (f *fakeProv) GetBool(key string) bool {
	return false
}

func (f *fakeProv) GetStringSlice(key string) []string {
	return nil
}

func (f *fakeProv) IsSet(key string) bool {
	return false
}

func (f *fakeProv) UnmarshalKey(key string, rawVal interface{}) error {
	return nil
}

func (f *fakeProv) ConfigFileUsed() string {
	return ""
}

func (f *fakeProv) GetPath(key string) string {
	return ""
}

func (f *fakeProv) TranslatePath(path string) string {
	return ""
}
