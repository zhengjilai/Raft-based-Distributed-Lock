FROM golang:1.12

RUN apt install -y software-properties-common git

# the main work_dir and other required code dirs
RUN mkdir -p /srv/gopath/srv/github.com/dlock_raft
ENV GOPATH /srv/gopath

# Dependencies (e.g. protobuf, yaml)
RUN mkdir -p /srv/gopath/src/google.golang.org/protobuf
RUN git clone https://github.com/protocolbuffers/protobuf-go.git /srv/gopath/src/google.golang.org/protobuf

RUN mkdir -p /srv/gopath/src/gopkg.in/
RUN git clone https://github.com/go-yaml/yaml.git /srv/gopath/src/gopkg.in/yaml.v2

RUN mkdir -p /srv/gopath/src/golang.org/x
RUN git clone https://github.com/golang/net.git /srv/gopath/src/golang.org/x/net
RUN git clone https://github.com/golang/sys.git /srv/gopath/src/golang.org/x/sys
RUN git clone https://github.com/golang/text.git /srv/gopath/src/golang.org/x/text

RUN mkdir -p /srv/gopath/src/github.com/golang
RUN git clone https://github.com/golang/protobuf.git /srv/gopath/src/github.com/golang/protobuf

# copy own codes into container
RUN cp ./protobuf /srv/gopath/src/github.com/dlock_raft/protobuf
RUN cp ./storage /srv/gopath/src/github.com/dlock_raft/storage
RUN cp ./node /srv/gopath/src/github.com/dlock_raft/node
RUN cp ./utils /srv/gopath/src/github.com/dlock_raft/utils

# mkdir for the config file dir
RUN mkdir -p /srv/gopath/src/github.com/dlock_raft/config
# mkdir for the var (persistent storage) dir
RUN mkdir -p /srv/gopath/src/github.com/dlock_raft/var

# entry point
# should expose two ports, one for peer, one for client
WORKDIR /srv/gopath/src/github.com/dlock_raft
ENTRYPOINT ["go", "run", "/srv/gopath/src/github.com/dlock_raft/start_node.go"]
