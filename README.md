# Raft-based-Distributed-Lock
A Golang implementation of Distributed Lock based on Raft consensus algorithm.

## Dependencies
### Local deployment dependencies
<h3 id="dependencies"></h3>
Basic requirements for local deployment of dlock_raft are listed as follows:

- Golang, version 1.12+
- protoc, version 3.0.0+
  + remember to compile protoc-gen-go in order to generate go files for grpc
  + optional, as we have already generated those grpc related codes for you.
- golang packages, including follows:
  + "golang.org/x/text"
  + "golang.org/x/net"
  + "golang.org/x/sys"
  + "gopkg.in/yaml.v2"
  + "github.com/golang/protobuf"
  + "github.com/segmentio/ksuid"
  + "google.golang.org/protobuf"
  + "google.golang.org/grpc"
  + "google.golang.org/genproto"

You can either obtain all these packages with `go get` or get their source codes from github with `git clone`. 

If you are still vague, please refer to `Dockerfile` for more details, 
where we solve all requirement problems for the image through simple shell scripts.

### Docker deployment dependencies
Another (recommended) way of deploying this module is to use docker.
Under this circumstance, the requirements are listed as follows:

- Docker, version 19.03+
- Docker-compose, version 1.22+

## Service Deployment

### Preparations

We provide two alternatives for deploying our system, 
respectively local deployment and docker deployment.
For either choice, first clone the repository in `$GOPATH/src/github.com` at your local machine.

```shell
mkdir -p $GOPATH/src/github.com && cd $GOPATH/src/github.com
git clone git@github.com:zhengjilai/Raft-based-Distributed-Lock.git
mv Raft-based-Distributed-Lock dlock_raft
```

Then, generate all grpc codes for the project automatically with `protoc` and `protoc-gen-go`.
This procedure is optional, since we have already generated those grpc codes for you.

```shell
cd $PROJECT_DIR
protoc --proto_path=. --go_out=plugins=grpc:$GOPATH/src ./protobuf/*.proto
```

Finally, we also export an environment variable dubbed `PROJECT_DIR` for convenience.

```shell
export PROJECT_DIR=$GOPATH/src/github.com/dlock_raft/
```

### Service local deployment 

For local deployment, first obtain all required packages listed in [Section of Dependencies](#dependencies).

Then, revise the config file (`$PROJECT_DIR/config/config.yaml`). 
You can refer to [Section of Configuration](#configuration) for more details. 

Finally, the distributed lock server can be started locally with the following shell script.

```shell
cd $PROJECT_DIR
go run start_node.go
```

### Service docker deployment 

First, build the node image with `docker build`.

```shell
cd $PROJECT_DIR
# for x86_64 architecture
docker build -t zhengjilai/raft-based-dlock:0.0.1 .
# for aarch64 (arm64v8) architecture
docker build -t zhengjilai/raft-based-dlock:0.0.1-arm64 -f Dockerfile-arm64v8 .
```

Then you should revise the config file (`$PROJECT_DIR/config/config.yaml`). 
You can refer to [Section of Configuration](#configuration) for more details. 

Finally, you can start the distributed lock service with docker-compose.
Remember to expose the same port in `docker-compose-local.yaml` 
as how `self_cli_address` is defined in `$PROJECT_DIR/config/config.yaml`.

```shell
cd $PROJECT_DIR
docker-compose -f docker-compose-local.yaml up
```

The service can be stopped with the following shell script.

```shell
cd $PROJECT_DIR
docker-compose -f docker-compose-local.yaml down
```

## Service Configuration
<h3 id="configuration"></h3>
A config file is required when starting the distributed lock node, 
both for local deployment and docker deployment.

### Location of config file

For **local deployment**, file config locates in `$PROJECT_DIR/config/config.yaml`.

For **docker deployment**, the file config should exist as `/srv/gopath/src/github.com/dlock_raft/config/config.yaml`
in the docker container.
Generally, we will use `-v` (docker run) or `volume` (docker-compose) 
to reflect an outside config file into the container.
See `docker-compose-local.yaml` if you are still vague about it.

### Contents in config file

In short, config file contains four parts, namely id, address, parameters and storage.

**Id configuration** determines the globally unique id for nodes.
A typical id configuration is shown as follows. `self_id` is the id of this specific node, 
and `peer_id` are the id of other nodes in the distributed lock system. 
The setting of id only have two restrictions. 
First, all ids are globally unique, and your `self_id` should exist in all other nodes' `peer_id` list.
Second, `0` is invalid for an id.

```yaml
# id configuration for node 1
id:
  self_id: 1
  peer_id:
    - 2
    - 3
# id configuration for node 2: self_id = 2, and peer_id = [1,3]
# id configuration for node 3: self_id = 3, and peer_id = [1,2]
```

**Address configuration** determines the address for rpc connection. 
A typical address configuration is shown as follows. 
`self_address` and `peer_address` are utilized for grpc connection between nodes in the cluster.
`self_cli_address` and `peer_cli_address` are utilized for connection between nodes and client 
(namely distributed lock acquirers). 
Similarly, all addresses should be globally unique, 
and your `self_xx` should exist in all other nodes' `peer_xx` list, just as how id configuration does.

```yaml
# a typical id configuration
network:
  self_address: "192.168.2.30:14005"
  self_cli_address: "192.168.2.30:24005"
  peer_address:
    - "192.168.0.2:14005"
    - "192.168.0.3:14005"
  peer_cli_address:
    - "192.168.0.2:24005"
    - "192.168.0.3:24005"
```

**Parameter configuration** determines some internal parameters for raft or dlock.
Generally you do not need to revise them. See comments if you want to revise certain parameters.

**Storage configuration** are simply the path for Entry database and system log. 
For `log_level`, the default is Info. 
However, you can select your wanted log level from Critical, Error, Warning, Notice, Info, Debug.

## Client API Introduction
We provide some simple client API in package `github.com/dlock_raft/api`. 
Our dlock provides basic functionalities of acquire, query and release. 
Besides, a mechanism of expire (or lease) is available to locks,  
and all acquirement for the same dlock will be accepted according to FIFO paradigm.

### API specifications

**DLock API Handler**. 
To operate on dlock, you should first generate an API handler as follows. 

```go
dlockClient := NewDLockRaftClientAPI()
// dlockClint := NewDLockRaftClientAPI(existingClientId)
```

If you do not specify a client id, the handler will automatically generate one for you. 
Client Id follows the format of "github.com/segmentio/ksuid".

**Acquire DLock**. A distributed lock can be acquired as follows. 

```go
ok := dlockClient.AcquireDLock(address, lockName, expire, timeout)
``` 

Here `address` follows the format of "ip:port", `lockName` is an arbitrary string set as the name of lock, 
`expire` is an int64 time duration for lock lease (unit: ms), 
`timeout` is the maximum block time for acquirement (optional, unit: ms).
`ok == true` iff the dlock is acquired successfully.

**Query DLock**. The current state of a distributed lock can be queried as follows.

```go
dlockInfo, ok := dlockClient.QueryDLock(address, lockName)
```

`dlockInfo` contains the current state of lock, 
including its owner, nonce, expire, timestamp of last acquirement, pending acquirement number, etc..
`ok == true` iff the dlock exists (either locking or released).
If the lock have been released, dlockInfo.Owner is an empty string.

**Release DLock**. A distributed lock can be released actively as follows. 
```go
ok := dlockClient.ReleaseDLock(address, lockName)
```

`ok == true` iff the lock is released in this release request. 
Also note that dlock will be released by the cluster automatically when its expires.

### Other notifications

- **DLock Expire**. 
All distributed locks must possess an expire (lease). 
Even if the lock owner does not release the lock actively, 
the service cluster will release the lock as soon as it expires.
Expire (lease) must be set when acquiring a dlock (unit: ms).

- **Acquirement Timeout**. 
When acquiring a distributed lock, the client may block as the lock is occupied, 
namely acquiring the lock continuously without quitting.
Acquirement timeout is the maximum blocking time before the client "give up".
Expire (lease) can be set optionally when acquiring a dlock (unit: ms), 2500 ms by default.

- **FIFO DLock Acquirement**.
Our service follows the paradigm of FIFO for acquirements.
For example, an occupied lock is being acquired by A, B, and C (ordered by their first acquiring time).
When the lock is released, A will definitely own the lock other than B and C,
 as long as A is still acquiring the lock.

- **Key-Value Storage API**. 
Client API for KV storage is also provided, including PutState, GetState, DelState. 
Refer to integrated test for their usage if you need them (smile).

## Integrated Experiments
We provide two integrated experiments, including a local deployment example with docker-compose
and a (simulated) real-life deployment example in a distributed environment.

### Local deployment example
<h3 id="localtest"></h3>

Local deployment test should be conducted with docker-compose.
All materials for local test are placed in `$PROJECT_DIR/experiments/local_test_3nodes`.

First, start 3 docker containers to compose a distributed lock.
```shell
cd $PROJECT_DIR/experiments/local_test_3nodes
make start
```

We provide some integrated tests in `$PROJECT_DIR/client/dlock_raft_client_API_test.go`
for this locally deployed cluster. You can trigger all of them with `go test`. 
Your local cluster is running normally if all integrated tests are passed.
```shell
go test github.com/dlock_raft/api
```

To stop the cluster, you can use `make stop`; 
to clean all existing system logs and Entries in database you can use `make clean`.

Note that in local test, we by default expose port 24005-24007 
for clients to communicate with those three dlock nodes (namely `self_cli_address`). 
Thus, the addressList in `$PROJECT_DIR/client/dlock_raft_client_API_test.go` is set as follows.
You can revise them when doing more complicated integrated tests.
```go
var addressList = [...]string {
	"0.0.0.0:24005",
	"0.0.0.0:24006",
	"0.0.0.0:24007",
}
```

### Distributed deployment example
Distributed deployment can be conducted with the following prerequisites.

- At least 3 remote machines are available and can communicate with each other. 
- All remote machines enables ssh no-passwd-login (PublicKeyAuthentication).
- All remote machines can expose at least two ports (for node-to-node connection and clisrv connection).
- Docker and docker-compose are available on remote machines.

All materials for local test are placed in `$PROJECT_DIR/experiments/distributed_tests`.

To conduct distributed test, first revise the shell variables in config `$PROJECT_DIR/experiments/distributed_tests/distributed_deploy.sh`, 
including the following network information: p2p_address, p2p_port, clisrv_address, clisrv_port, etc..
Please refer to the shell script `distributed_deploy.sh` for more details.

Then, you can generate all materials for all service nodes, and scp them on all remote machines 
with the shell script we provide.
```shell
cd $PROJECT_DIR/experiments/distributed_tests
./distributed_deploy.sh genAllMat && ./distributed_deploy.sh scpAllMat
```

Finally, after you have copied all Materials on remote machines, start or stop all dlock service as follows.
```shell
cd $PROJECT_DIR/experiments/distributed_tests
# start dlock service on all nodes
./distributed_deploy.sh startAllService
# stop dlock service on all nodes
./distributed_deploy.sh stopAllService
```

You can test the state of the cluster with `go test github.com/dlock_raft/api`, 
just as what we have done in [Local Deployment Test](#localtest). 
Do not forget to revise server addresses in `$PROJECT_DIR/client/dlock_raft_client_API_test.go`.


## Contributors

- [Jilai Zheng](https://github.com/zhengjilai)