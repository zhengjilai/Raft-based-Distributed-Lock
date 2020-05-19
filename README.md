## Raft-based-Distributed-Lock
A Golang implementation of Distributed Lock based on Raft consensus algorithm.

### Usage
- Clone the project in GOPATH at your local machine.
```shell
mkdir -p $GOPATH/src/github.com
cd $GOPATH/src/github.com
git clone git@github.com:zhengjilai/Raft-based-Distributed-Lock.git
mv Raft-based_Distributed_Lock dlock_raft
```

- Generate grpc codes for raft.
```shell
cd dlock_raft
protoc --proto_path=. --go_out=plugins=grpc:$GOPATH/src ./protobuf/*.proto
```
