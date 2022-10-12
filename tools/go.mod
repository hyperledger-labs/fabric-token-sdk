module tools

go 1.18

replace (
	github.com/fsouza/go-dockerclient => github.com/fsouza/go-dockerclient v1.4.1
	github.com/go-kit/kit => github.com/go-kit/kit v0.7.0
	github.com/hyperledger/fabric => github.com/hyperledger/fabric v1.4.0-rc1.0.20210722174351-9815a7a8f0f7
	github.com/hyperledger/fabric-protos-go => github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354
	go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20181228115726-23731bf9ba55
)

require (
	github.com/client9/misspell v0.3.4
	github.com/fzipp/gocyclo v0.6.0
	github.com/google/addlicense v1.0.0
	github.com/gordonklaus/ineffassign v0.0.0-20210914165742-4cc7213b9bc8
	github.com/maxbrunsfeld/counterfeiter/v6 v6.5.0
	github.com/onsi/ginkgo/v2 v2.1.4
	golang.org/x/tools v0.1.11
	google.golang.org/protobuf v1.27.1
	honnef.co/go/tools v0.3.2
)

require (
	github.com/BurntSushi/toml v0.4.1 // indirect
	github.com/bmatcuk/doublestar/v4 v4.0.2 // indirect
	github.com/go-task/slim-sprig v0.0.0-20210107165309-348f09dbbbc0 // indirect
	github.com/google/go-cmp v0.5.6 // indirect
	github.com/google/pprof v0.0.0-20210407192527-94a9f03dee38 // indirect
	github.com/stretchr/testify v1.7.1-0.20210116013205-6990a05d54c2 // indirect
	golang.org/x/exp/typeparams v0.0.0-20220218215828-6cf2b201936e // indirect
	golang.org/x/mod v0.6.0-dev.0.20220419223038-86c51ed26bb4 // indirect
	golang.org/x/sync v0.0.0-20220601150217-0de741cfad7f // indirect
	golang.org/x/sys v0.0.0-20220615213510-4f61da869c0c // indirect
	golang.org/x/xerrors v0.0.0-20220517211312-f3a8303e98df // indirect
	gopkg.in/yaml.v3 v3.0.0-20210107192922-496545a6307b // indirect
)
