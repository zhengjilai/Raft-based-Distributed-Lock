FROM golang:1.12

RUN apt install -y git

# the main work_dir and other required code dirs
RUN mkdir -p /srv/gopath/srv/github.com/dlock_raft
ENV GOPATH /srv/gopath

# All Golang packages (e.g. protobuf, yaml)
RUN mkdir -p /srv/gopath/src/google.golang.org/
RUN git clone https://github.com/protocolbuffers/protobuf-go.git /srv/gopath/src/google.golang.org/protobuf
RUN git clone https://github.com/grpc/grpc-go.git /srv/gopath/src/google.golang.org/grpc
RUN git clone https://github.com/google/go-genproto /srv/gopath/src/google.golang.org/genproto

RUN mkdir -p /srv/gopath/src/gopkg.in/
RUN git clone https://github.com/go-yaml/yaml.git /srv/gopath/src/gopkg.in/yaml.v2

RUN mkdir -p /srv/gopath/src/golang.org/x
RUN git clone https://github.com/golang/net.git /srv/gopath/src/golang.org/x/net
RUN git clone https://github.com/golang/sys.git /srv/gopath/src/golang.org/x/sys
RUN git clone https://github.com/golang/text.git /srv/gopath/src/golang.org/x/text

RUN mkdir -p /srv/gopath/src/github.com/golang
RUN git clone https://github.com/golang/protobuf.git /srv/gopath/src/github.com/golang/protobuf

RUN mkdir -p /srv/gopath/src/github.com/segmentio
RUN git clone https://github.com:segmentio/ksuid.git /srv/gopath/src/github.com/segmentio/ksuid

# copy own codes into container
COPY ./protobuf /srv/gopath/src/github.com/dlock_raft/protobuf
COPY ./storage /srv/gopath/src/github.com/dlock_raft/storage
COPY ./node /srv/gopath/src/github.com/dlock_raft/node
COPY ./utils /srv/gopath/src/github.com/dlock_raft/utils
COPY start_node.go /srv/gopath/src/github.com/dlock_raft/start_node.go

# mkdir for the config file dir
RUN mkdir -p /srv/gopath/src/github.com/dlock_raft/config
# mkdir for the var (persistent storage) dir
RUN mkdir -p /srv/gopath/src/github.com/dlock_raft/var

# entry point
# should expose two ports, one for peer, one for client
WORKDIR /srv/gopath/src/github.com/dlock_raft
ENTRYPOINT ["go", "run", "start_node.go"]
