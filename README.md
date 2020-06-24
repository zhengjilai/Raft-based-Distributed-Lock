# Raft-based-Distributed-Lock
A Golang implementation of Distributed Lock based on Raft consensus algorithm.

## Dependencies
### Local Deployment Dependencies
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

### Docker Deployment Dependencies
Another (recommended) way of deploying this module is to use docker.
Under this circumstance, the requirements are listed as follows:

- Docker, version 19.03+
- Docker-compose, version 1.22+

## Deployment
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

### Local Deployment 

For local deployment, first obtain all required packages listed in [Section of Dependencies](#dependencies).

Then, revise the config file (`$PROJECT_DIR/config/config.yaml`). 
You can refer to the Section of Config for more details. 

Finally, the distributed lock server can be started locally with the following shell script.

```shell
cd $PROJECT_DIR
go run start_node.go
```

### Docker Deployment 

First, build the node image with `docker build`.

```shell
cd $PROJECT_DIR
docker build -t dlock_raft:0.0.1 .
```

Then you should revise the config file (`$PROJECT_DIR/config/config.yaml`). 
You can refer to the section of config for more details. 

Finally, you can start the distributed lock service with `docker run`.
Remember to expose the same port in `docker-compose-local.yaml` 
as `self_cli_address` defines in `$PROJECT_DIR/config/config.yaml`.

```shell
cd $PROJECT_DIR
docker-compose -f docker-compose-local.yaml up
```

The service can be stopped with the following shell script.

```shell
cd $PROJECT_DIR
docker-compose -f docker-compose-local.yaml down
```

## Configuration

A config file is required when starting the distributed lock node, 
both for local deployment and docker deployment.

### Location of config file

For local deployment, file config locates in `$PROJECT_DIR/config/config.yaml`.

For docker deployment, the file config should exist as `/srv/gopath/src/github.com/dlock_raft/config/config.yaml`
in the docker container.
Generally, we will use `-v` (docker run) or `volume` (docker-compose) 
to reflect an outside config file into the container.
See `docker-compose-local.yaml` if you are still vague about it.

### Content if config file

In short, config file contains four parts, namely id, address, parameters and storage.

A typical id is shown as follows. `self_id` is the id of this specific node, 
and `peer_id` are the id of other nodes in the distributed lock system. 
The setting of id only have two restrictions. 
First, all ids are globally unique, and your `self_id` should exist in all other nodes' `peer_id` list.
Second, `0` is invalid for an id.
```
id:
  self_id: 1
  peer_id:
    - 2
    - 3
```


- "address" are relevant to the network connection of your distributed lock cluster, 
while paths in "storage" determines the place for logs and logEntry database.
On the other hand, you do not need to revise "parameters" generally.



## Experiments
We provide two test templates, including a local deployment example with docker-compose
and a (simulated) real-life deployment example in a distributed environment.

### Local Deployment Example

Local deployment test should be conducted with docker-compose.
All materials for local test is placed in `SPROJECT_DIR/experiments/local_test_3nodes`.


### Distributed Deployment Example
2