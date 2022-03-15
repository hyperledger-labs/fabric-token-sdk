module github.com/hyperledger-labs/fabric-token-sdk

go 1.16

replace (
	github.com/fsouza/go-dockerclient => github.com/fsouza/go-dockerclient v1.4.1
	github.com/go-kit/kit => github.com/go-kit/kit v0.7.0
	github.com/hyperledger/fabric => github.com/hyperledger/fabric v1.4.0-rc1.0.20210722174351-9815a7a8f0f7
	github.com/hyperledger/fabric-protos-go => github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354
	go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20181228115726-23731bf9ba55
)

require (
	github.com/IBM/idemix v0.0.0-20220113150823-80dd4cb2d74e
	github.com/IBM/mathlib v0.0.0-20220112091634-0a7378db6912
	github.com/containerd/containerd v1.5.5 // indirect
	github.com/dgraph-io/badger/v3 v3.2103.2
	github.com/golang/protobuf v1.5.2
	github.com/hashicorp/go-uuid v1.0.2
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hyperledger-labs/fabric-smart-client v0.0.0-20220302104532-f1f2fa782bea
	github.com/hyperledger/fabric v1.4.0-rc1.0.20220128025611-fad7f691a967
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20210718160520-38d29fabecb9
	github.com/hyperledger/fabric-protos-go v0.0.0-20210911123859-041d13f0980c
	github.com/json-iterator/go v1.1.10
	github.com/libp2p/go-libp2p-core v0.3.0
	github.com/maxbrunsfeld/counterfeiter/v6 v6.3.0
	github.com/moby/term v0.0.0-20210619224110-3f7ff695adc6 // indirect
	github.com/onsi/ginkgo v1.16.5
	github.com/onsi/gomega v1.16.0
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.1-0.20210116013205-6990a05d54c2
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	github.com/thedevsaddam/gojsonq v2.3.0+incompatible
	go.uber.org/atomic v1.7.0
	go.uber.org/zap v1.16.0
	golang.org/x/sys v0.0.0-20220307203707-22a9840ba4d7 // indirect
	gopkg.in/yaml.v2 v2.4.0
)
