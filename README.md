# Raft-based-Distributed-Lock
A Golang implementation of Distributed Lock based on Raft consensus algorithm.

## Dependencies
### Local Deployment Dependencies
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

If you are still ambiguous, please refer to `Dockerfile` for more details, 
where we solve all requirement problems for the image through simple shell scripts.

### Docker Deployment Dependencies
Another (recommended) way of deploying this module is to use docker.
Under this circumstance, the requirements are listed as follows:

- Docker, version 19.03+
- Docker-compose, version 1.22+

## Deployment
We provide two alternatives for deploying our system, 
respectively local deployment and docker deployment.
For either choice, Clone the repository according to `$GOPATH` at your local machine.
```shell
mkdir -p $GOPATH/src/github.com && cd $GOPATH/src/github.com
git clone git@github.com:zhengjilai/Raft-based-Distributed-Lock.git
mv Raft-based_Distributed_Lock dlock_raft
```

### Local Deployment 

First, generate all grpc codes for the project automatically with `protoc` and `protoc-gen-go`.
This procedure is optional, since we have already generated those grpc codes for you.
```shell
cd $GOPATH/src/github.com/dlock_raft
protoc --proto_path=. --go_out=plugins=grpc:$GOPATH/src ./protobuf/*.proto
```

### Docker Deployment 

First, build the node image with `docker build`.
```shell
cd $GOPATH/src/github.com/dlock_raft
docker build -t dlock_raft:0.0.1 .
```

## Experiments
We provide two test templates, including a local deployment example with docker-compose 
and a (simulated) real-life deployment example in a distributed environment.

### Local Deployment Example
1
### Distributed Deployment Example
2