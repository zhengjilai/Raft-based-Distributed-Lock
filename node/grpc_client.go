package node
// the gprc-based client, for P2P transport between nodes

import (
	pb "github.com/dlock_raft/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
)

type GrpcClientImpl struct {

	// the connect peer address, format ip:port
	peerAddress string
	conn *grpc.ClientConn
	// the NodeRef Object
	NodeRef *Node
}

func NewGrpcClient(address string, node *Node) *GrpcClientImpl {

	// construct the feedback object
	grpcClient := new(GrpcClientImpl)
	grpcClient.peerAddress = address
	grpcClient.conn = nil
	grpcClient.NodeRef = node
	return grpcClient
}

func (gc *GrpcClientImpl) StartConnection() error {
	// dial the server without secure option
	conn, err := grpc.Dial(gc.peerAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}
	gc.conn = conn
	return nil
}

func (gc *GrpcClientImpl) GetConnectionState() connectivity.State {
	if gc.conn != nil {
		return gc.conn.GetState()
	}
	return connectivity.Idle
}

func (gc *GrpcClientImpl) IsAvailable() bool {
	if gc.conn != nil {
		return gc.conn.GetState() == connectivity.Ready
	}
	return false
}

func (gc *GrpcClientImpl) ReConnect() error {
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
	}
	return nil
}

func (gc *GrpcClientImpl) SendGrpcAppendEntries(request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		gc.NodeRef.NodeLogger.Infof("Current TCP state is %s\n", gc.conn.GetState())
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.\n", gc.peerAddress)
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
	gc.NodeRef.NodeLogger.Infof("Begin to send AppendEntry to %s.\n", gc.peerAddress)
	response, err := clientAppendEntry.AppendEntriesService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcClientImpl) SendGrpcCandidateVotes(request *pb.CandidateVotesRequest) (*pb.CandidateVotesResponse, error){

	// test whether connection still exists
	if !gc.IsAvailable() {
		gc.NodeRef.NodeLogger.Infof("Current TCP state is %s\n", gc.conn.GetState())
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.\n", gc.peerAddress)
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
	gc.NodeRef.NodeLogger.Infof("Begin to send CandidateVote to %s.\n", gc.peerAddress)
	response, err := clientCandidateVote.CandidateVotesService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcClientImpl) SendGrpcRecoverEntries(request *pb.RecoverEntriesRequest) (*pb.RecoverEntriesResponse, error){

	// test whether connection still exists
	if !gc.IsAvailable() {
		gc.NodeRef.NodeLogger.Infof("Current TCP state is %s\n", gc.conn.GetState())
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.\n", gc.peerAddress)
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
	gc.NodeRef.NodeLogger.Infof("Begin to sen RecoverEntries to %s.\n", gc.peerAddress)
	response, err := clientRecoverEntries.RecoverEntriesService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}
	return response, nil
}

//func main() {
//	var request chan string = make(chan string, 100)
//	var result chan string = make(chan string, 100)
//	conn := newGrpcConn("127.0.0.1:30051", request, result)
//
//	go func() {
//		for {
//			conn.SendDta()
//		}
//	}()
//
//	go func() {
//		var index int
//		for {
//			request <- fmt.Sprintf("%d", index)
//			index++
//			time.Sleep(time.Second)
//		}
//	}()
//
//	for {
//		v := <-result
//		fmt.Println(" result ", v)
//	}
//}

