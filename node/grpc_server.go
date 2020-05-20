package node

// the gprc-based server, for P2P transport between nodes

import (
	"errors"
	pb "github.com/dlock_raft/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"strings"
)

var GRPCServerAddressError = errors.New("dlock_raft.gprc_server: " +
	"the listening address is wrong, should have format ip:port")
var GRPCServerDeadError = errors.New("dlock_raft.gprc_server: " +
	"the node state is Dead, should start it")

type GrpcServerImpl struct {

	// the actual grpc server object
	grpcServer *grpc.Server
	// the NodeRef Object
	NodeRef *Node
}

func NewGrpcServer(node *Node) (*GrpcServerImpl, error){
	grpcServer := new(GrpcServerImpl)
	grpcServer.NodeRef = node
	return grpcServer, nil
}

func (gs *GrpcServerImpl) AppendEntriesService(ctx context.Context,
	request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {

	// lock before doing anything
	gs.NodeRef.mutex.Lock()
	defer gs.NodeRef.mutex.Unlock()

	// if node state is Dead, stop doing anything
	if gs.NodeRef.NodeContextInstance.NodeState == Dead {
		return nil, GRPCServerDeadError
	}
	gs.NodeRef.NodeLogger.Infof("Begin to precess AppendEntries request, %+v.", request)

	response := &pb.AppendEntriesResponse{
		Term:             0,
		NodeId:           0,
		PrevEntryIndex:   0,
		PrevEntryTerm:    0,
		CommitEntryIndex: 0,
	}

	// the remote term exceeds local term, should become follower
	if request.Term > gs.NodeRef.NodeContextInstance.CurrentTerm {
		gs.NodeRef.NodeLogger.Infof("AppendEntry term %d is greater than current term %d.",
			request.Term, gs.NodeRef.NodeContextInstance.CurrentTerm)
		gs.NodeRef.BecomeFollower()
	}
	//
	if request.Term >
	request.EntryList[]

	return , nil
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
	address := gs.NodeRef.NodeConfigInstance.Network.SelfAddress
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
