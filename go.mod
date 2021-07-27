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
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/consensys/gurvy v0.3.9-0.20210209011448-37644c45f955
	github.com/dgraph-io/badger/v3 v3.2011.1
	github.com/golang/protobuf v1.5.0
	github.com/google/addlicense v0.0.0-20210729153508-ef04bb38a16b // indirect
	github.com/gordonklaus/ineffassign v0.0.0-20210729092907-69aca2aeecd0 // indirect
	github.com/hyperledger-labs/fabric-smart-client v0.0.0-20210729224643-9c6c674c1198
	github.com/hyperledger/fabric v1.4.0-rc1.0.20210722174351-9815a7a8f0f7
	github.com/hyperledger/fabric-amcl v0.0.0-20200424173818-327c9e2cf77a
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200424173110-d7076418f212
	github.com/hyperledger/fabric-protos-go v0.0.0-20210720123151-f0dc3e2a0871
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.10.3
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	go.uber.org/atomic v1.7.0
	gopkg.in/yaml.v2 v2.4.0
)
