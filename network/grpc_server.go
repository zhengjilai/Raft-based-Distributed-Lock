package network

// the gprc-based server, for P2P transport between nodes

import (
	"errors"
	"github.com/dlock_raft/node"
	pb "github.com/dlock_raft/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"strings"
)

var GRPCServerAddressError = errors.New("dlock_raft.gprc_server: " +
	"the listening address is wrong, should have format ip:port")

type GrpcServerImpl struct {

	// the actual grpc server object
	grpcServer *grpc.Server
	// the node Object
	node *node.Node
}

func NewGrpcServer(node *node.Node) (*GrpcServerImpl, error){
	grpcServer := new(GrpcServerImpl)
	grpcServer.node = node
	return grpcServer, nil
}

func (gs *GrpcServerImpl) AppendEntriesService(ctx context.Context,
	request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {

	return &pb.AppendEntriesResponse{
		Term:             0,
		NodeId:           0,
		PrevEntryIndex:   0,
		PrevEntryTerm:    0,
		CommitEntryIndex: 0,
	}, nil
}

func (gs *GrpcServerImpl) CandidateVotesService(ctx context.Context,
	request *pb.CandidateVotesRequest) (*pb.CandidateVotesResponse, error) {

	return &pb.CandidateVotesResponse{
		Term:     0,
		NodeId:   0,
		Accepted: false,
	}, nil
}

func (gs *GrpcServerImpl) RecoverEntriesService(ctx context.Context,
	request *pb.RecoverEntriesRequest) (*pb.RecoverEntriesResponse, error) {

	return &pb.RecoverEntriesResponse{
		Term:           0,
		NodeId:         0,
		PrevEntryIndex: 0,
		PrevEntryTerm:  0,
		Recovered:      false,
		EntryList:      nil,
	}, nil
}

// should input the self
func (gs *GrpcServerImpl) StartService() error {

	// get the listing address
	address := gs.node.NodeConfig.Network.SelfAddress
	splittedAddress := strings.Split(address, ":")
	if len(splittedAddress) != 2 {
		return GRPCServerAddressError
	}

	// listening at a specific port
	listener, err := net.Listen("tcp", ":" + splittedAddress[1])
	if err != nil {
		return err
	}
	// start a new grpc server
	gs.grpcServer = grpc.NewServer()
	pb.RegisterRaftRPCServerServer(gs.grpcServer, gs)

	// start service
	err = gs.grpcServer.Serve(listener)
	if err != nil {
		return err
	}
	return nil
}
