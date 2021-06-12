module github.com/hyperledger-labs/fabric-token-sdk

go 1.14

replace (
	github.com/fsouza/go-dockerclient => github.com/fsouza/go-dockerclient v1.4.1
	github.com/go-kit/kit => github.com/go-kit/kit v0.7.0
	github.com/golang/protobuf => github.com/golang/protobuf v1.3.3
	github.com/spf13/viper => github.com/spf13/viper v0.0.0-20150908122457-1967d93db724
	go.etcd.io/etcd => go.etcd.io/etcd v0.5.0-alpha.5.0.20181228115726-23731bf9ba55
	google.golang.org/genproto => google.golang.org/genproto v0.0.0-20190819201941-24fa4b261c55
	google.golang.org/grpc => google.golang.org/grpc v1.29.1
)

require (
	github.com/alecthomas/template v0.0.0-20190718012654-fb15b899a751
	github.com/consensys/gurvy v0.3.9-0.20210209011448-37644c45f955
	github.com/dgraph-io/badger/v3 v3.2011.1
	github.com/golang/protobuf v1.5.2
	github.com/hyperledger-labs/fabric-smart-client v0.0.0-20210609074109-b623df1c373b
	github.com/hyperledger/fabric v1.4.0-rc1.0.20200930182727-344fda602252
	github.com/hyperledger/fabric-amcl v0.0.0-20200424173818-327c9e2cf77a
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20200424173110-d7076418f212
	github.com/hyperledger/fabric-protos-go v0.0.0-20200506201313-25f6564b9ac4
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.10.1
	github.com/pkg/errors v0.9.1
	github.com/spf13/cobra v1.1.1
	github.com/spf13/viper v1.7.0
	github.com/stretchr/testify v1.7.0
	github.com/tedsuo/ifrit v0.0.0-20191009134036-9a97d0632f00
	go.uber.org/atomic v1.7.0
	golang.org/x/sys v0.0.0-20210611083646-a4fc73990273 // indirect
	golang.org/x/tools v0.1.3 // indirect
	google.golang.org/grpc v1.36.1 // indirect
	google.golang.org/protobuf v1.26.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
)
