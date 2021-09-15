module tcc

go 1.16

replace (
	github.com/hyperledger-labs/fabric-token-sdk => github.com/hyperledger-labs/fabric-token-sdk v0.0.0-20210909113825-373ae3c653c2
	github.com/hyperledger/fabric-protos-go => github.com/hyperledger/fabric-protos-go v0.0.0-20201028172056-a3136dde2354
)

exclude github.com/hyperledger-labs/fabric-token-sdk v0.0.0

require (
	github.com/hyperledger-labs/fabric-token-sdk v0.0.0-20210805132955-7e398f7c34d7
	github.com/hyperledger/fabric-chaincode-go v0.0.0-20210718160520-38d29fabecb9
	github.com/moby/sys/mount v0.2.0 // indirect
)
