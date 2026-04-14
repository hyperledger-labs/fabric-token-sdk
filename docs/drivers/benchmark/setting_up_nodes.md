# Setting up 2 nodes Benchmarking on 2 Severs

To setup AWS EC2 nodes See: [AWS Setup 2 Machines](../../../token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/aws_setup_2_machines.md)

Architecture: 
1. Machine 1 is the server 
2. Machine 2 is the client sending to the server and gathering the metrics

Setup: 
1. Add ssh pubkey of server to client `known_hosts`
2. Start the server:

```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/
GOGC=10000 go run ./server/
```
3. In the client, Rsync the node data from the server:

```bash
cd token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/

rsync -avz MyServerName:/.../fabric-token-sdk/token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/out/ ./out
``` 
4. Replace the IP:
```bash
sed 's#127.0.0.1#123.456.789#g' ./out/testdata/fsc/nodes/test-node.0/client-config.yaml -i
```
5. Start client and tee output to file

```bash
GOGC=10000 nohup go run ./client/ -benchtime=30s -count=5 -workloads=transfer-service -cpu=1,2,4,8,16,32,48,64 -numConn=1,2,4,8 2>&1 | tee out.txt &
```

Running Local Benchmark:

```bash
GOGC=10000 go test ./token/core/zkatdlog/nogh/v1/validator/bench/transfer_service/ -run ^$ -ben
ch=BenchmarkLocalTransferService -benchtime=30s -count=5 -cpu=1,4,8,16,32,64
```
Other benchmarks:
```
GOGC=10000 go test -bench=BenchmarkAPIGRPC -benchtime=30s -count=5 -cpu=32 2>&1
```


