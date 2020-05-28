# Raft-based-Distributed-Lock
A Golang implementation of Distributed Lock based on Raft consensus algorithm.

## Dependencies
### Local Deployment
The basic requirements for the local deployment of dlock_raft are listed as follows:

- git
- Golang, version 1.12+
- protoc, version 3.0.0+
  + remember to compile protoc-gen-go in order to generate go files for grpc
- golang packages, including follows:
  + "golang.org/x/text"
  + "golang.org/x/net"
  + "golang.org/x/sys"
  + "gopkg.in/yaml.v2"
  + "github.com/golang/protobuf"
  + "google.golang.org/protobuf"
  + "google.golang.org/grpc"
  + "google.golang.org/genproto"

You can obtain all these packages and even their source codes from github. 

If you are still ambiguous, please refer to `Dockerfile` for more details.

### Docker Deployment
Another (recommended) way of deploying this module is to use docker.
Under this circumstance, the requirements are listed as follows:

- Docker, version 19.03+
- Docker-compose, version 1.22+

## Usage
1. Clone the project in GOPATH at your local machine.
```shell
mkdir -p $GOPATH/src/github.com
cd $GOPATH/src/github.com
git clone git@github.com:zhengjilai/Raft-based-Distributed-Lock.git
mv Raft-based_Distributed_Lock dlock_raft
```

### Local Machine Usage

1. Generate grpc codes for raft (optional, as we have already generated those grpc codes for you).
```shell
cd dlock_raft
protoc --proto_path=. --go_out=plugins=grpc:$GOPATH/src ./protobuf/*.proto
```

### Docker Usage
1. Build the image with docker
```shell
docker build -t dlock_raft:0.0.1 .
```


## Experiments
We provide two test templates, including a local deployment example with docker-compose 
and a (simulated) real-life deployment example in a distributed environment.

### Local Deployment Example
1
### Distributed Deployment Example
2