package node
// the gprc-based client, for P2P transport between nodes

import (
	pb "github.com/dlock_raft/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
)

type GrpcP2PClientImpl struct {

	// the connect peer address, format ip:port
	peerAddress string
	conn *grpc.ClientConn
	// the NodeRef Object
	NodeRef *Node
}

func NewGrpcP2PClientImpl(address string, node *Node) *GrpcP2PClientImpl {

	// construct the feedback object
	grpcClient := new(GrpcP2PClientImpl)
	grpcClient.peerAddress = address
	grpcClient.conn = nil
	grpcClient.NodeRef = node
	return grpcClient
}

func (gc *GrpcP2PClientImpl) StartConnection() error {
	// dial the server without secure option
	conn, err := grpc.Dial(gc.peerAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}
	gc.conn = conn
	return nil
}

func (gc *GrpcP2PClientImpl) GetConnectionState() connectivity.State {
	if gc.conn != nil {
		return gc.conn.GetState()
	}
	return connectivity.Idle
}

func (gc *GrpcP2PClientImpl) IsAvailable() bool {
	if gc.conn != nil {
		gc.NodeRef.NodeLogger.Infof("Current TCP state is %s", gc.conn.GetState())
		return gc.conn.GetState() == connectivity.Ready
	}
	gc.NodeRef.NodeLogger.Infof("P2P Connection with %s has not been initialized.", gc.peerAddress)
	return false
}

func (gc *GrpcP2PClientImpl) ReConnect() error {
	if gc.conn != nil && gc.conn.GetState() == connectivity.TransientFailure {
		err := gc.conn.Close()
		if err != nil {
			return err
		}
		conn, err := grpc.Dial(gc.peerAddress, grpc.WithInsecure())
		if err != nil {
			return err
		}
		gc.conn = conn
	} else if gc.conn == nil {
		return gc.StartConnection()
	}
	return nil
}

func (gc *GrpcP2PClientImpl) SendGrpcAppendEntries(request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.", gc.peerAddress)
			return nil, err
		}
	}

	// register grpc client
	clientAppendEntry := pb.NewRaftRPCServerClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.NodeRef.NodeConfigInstance.Parameters.AppendEntriesTimeout) * time.Millisecond)
	defer cancel()

	// handle response of AppendEntry
	gc.NodeRef.NodeLogger.Infof("Begin to send AppendEntry %+v to %s.", request, gc.peerAddress)
	response, err := clientAppendEntry.AppendEntriesService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcP2PClientImpl) SendGrpcCandidateVotes(request *pb.CandidateVotesRequest) (*pb.CandidateVotesResponse, error){

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.", gc.peerAddress)
			return nil, err
		}
	}

	// register grpc client
	clientCandidateVote := pb.NewRaftRPCServerClient(gc.conn)

	// set candidate vote timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.NodeRef.NodeConfigInstance.Parameters.MaxWaitTimeCandidate) * time.Millisecond)
	defer cancel()

	// handle response
	gc.NodeRef.NodeLogger.Infof("Begin to send CandidateVote %+v to %s.", request, gc.peerAddress)
	response, err := clientCandidateVote.CandidateVotesService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcP2PClientImpl) SendGrpcRecoverEntries(request *pb.RecoverEntriesRequest) (*pb.RecoverEntriesResponse, error){

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.", gc.peerAddress)
			return nil, err
		}
	}

	// register grpc client
	clientRecoverEntries := pb.NewRaftRPCServerClient(gc.conn)

	// set candidate vote timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.NodeRef.NodeConfigInstance.Parameters.AppendEntriesTimeout) * time.Millisecond)
	defer cancel()

	// handle response
	gc.NodeRef.NodeLogger.Infof("Begin to sen RecoverEntries %+v to %s.", request, gc.peerAddress)
	response, err := clientRecoverEntries.RecoverEntriesService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.", err)
		return nil, err
	}
	return response, nil
}

