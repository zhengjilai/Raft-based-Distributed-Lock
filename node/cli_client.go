package node
// the gprc-based client, for client-server transport between client and node

import (
	pb "github.com/dlock_raft/protobuf"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
)

type GrpcCliSrvClientImpl struct {

	// the connect peer address, format ip:port
	nodeAddress string
	conn *grpc.ClientConn
	// the NodeRef Object
	NodeRef *Node
}

func NewGrpcCliSrvClientImpl(address string, node *Node) *GrpcCliSrvClientImpl {

	// construct the feedback object
	grpcClient := new(GrpcCliSrvClientImpl)
	grpcClient.nodeAddress = address
	grpcClient.conn = nil
	grpcClient.NodeRef = node
	return grpcClient
}

func (gc *GrpcCliSrvClientImpl) StartConnection() error {
	// dial the server without secure option
	conn, err := grpc.Dial(gc.nodeAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}
	gc.conn = conn
	return nil
}

func (gc *GrpcCliSrvClientImpl) GetConnectionState() connectivity.State {
	if gc.conn != nil {
		return gc.conn.GetState()
	}
	return connectivity.Idle
}

func (gc *GrpcCliSrvClientImpl) IsAvailable() bool {
	if gc.conn != nil {
		gc.NodeRef.NodeLogger.Infof("Current TCP state is %s", gc.conn.GetState())
		return gc.conn.GetState() == connectivity.Ready
	}
	gc.NodeRef.NodeLogger.Infof("P2P Connection with %s has not been initialized.", gc.nodeAddress)
	return false
}

func (gc *GrpcCliSrvClientImpl) ReConnect() error {
	if gc.conn != nil && gc.conn.GetState() == connectivity.TransientFailure {
		err := gc.conn.Close()
		if err != nil {
			return err
		}
		conn, err := grpc.Dial(gc.nodeAddress, grpc.WithInsecure())
		if err != nil {
			return err
		}
		gc.conn = conn
	} else if gc.conn == nil {
		return gc.StartConnection()
	}
	return nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcPutState(request *pb.ClientPutStateKVRequest) (*pb.ClientPutStateKVResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientAppendEntry := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.NodeRef.NodeConfigInstance.Parameters.StateChangeTimeout + 100) * time.Millisecond)
	defer cancel()

	// handle response of AppendEntry
	gc.NodeRef.NodeLogger.Infof("Begin to send PutState %+v to %s.", request, gc.nodeAddress)
	response, err := clientAppendEntry.PutStateKVService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcDelState(request *pb.ClientDelStateKVRequest) (*pb.ClientDelStateKVResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientAppendEntry := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.NodeRef.NodeConfigInstance.Parameters.StateChangeTimeout + 100) * time.Millisecond)
	defer cancel()

	// handle response of AppendEntry
	gc.NodeRef.NodeLogger.Infof("Begin to send DelState %+v to %s.", request, gc.nodeAddress)
	response, err := clientAppendEntry.DelStateKVService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcGetState(request *pb.ClientGetStateKVRequest) (*pb.ClientGetStateKVResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			gc.NodeRef.NodeLogger.Errorf("Reconnect TCP to %s fails.", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientAppendEntry := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.NodeRef.NodeConfigInstance.Parameters.StateChangeTimeout + 100) * time.Millisecond)
	defer cancel()

	// handle response of AppendEntry
	gc.NodeRef.NodeLogger.Infof("Begin to send GetState %+v to %s.", request, gc.nodeAddress)
	response, err := clientAppendEntry.GetStateKVService(ctx, request)
	if err != nil {
		gc.NodeRef.NodeLogger.Errorf("Reconnect TCP fails %s.", err)
		return nil, err
	}

	return response, nil
}

