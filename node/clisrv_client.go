package node
// the gprc-based client, for client-server transport between client and node

import (
	"fmt"
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
	timeout uint32
}

func NewGrpcCliSrvClientImpl(address string, timeout uint32) *GrpcCliSrvClientImpl {

	// construct the feedback object
	grpcClient := new(GrpcCliSrvClientImpl)
	grpcClient.nodeAddress = address
	grpcClient.conn = nil
	grpcClient.timeout = timeout
	return grpcClient
}

func (gc *GrpcCliSrvClientImpl) SetTimeout(timeout uint32) {
	gc.timeout = timeout
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
		fmt.Printf("Current TCP state is %s.\n", gc.conn.GetState())
		return gc.conn.GetState() == connectivity.Ready
	}
	fmt.Printf("P2P Connection with %s has not been initialized.\n", gc.nodeAddress)
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

func (gc *GrpcCliSrvClientImpl) SendGrpcPutState(
	request *pb.ClientPutStateKVRequest) (*pb.ClientPutStateKVResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			fmt.Printf("Reconnect TCP to %s fails.\n", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientRPCOutside := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.timeout) * time.Millisecond)
	defer cancel()

	// handle response
	fmt.Printf("Begin to send PutState %+v to %s.\n", request, gc.nodeAddress)
	response, err := clientRPCOutside.PutStateKVService(ctx, request)
	if err != nil {
		fmt.Printf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcDelState(
	request *pb.ClientDelStateKVRequest) (*pb.ClientDelStateKVResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			fmt.Printf("Reconnect TCP to %s fails.\n", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientRPCOutside := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.timeout) * time.Millisecond)
	defer cancel()

	// handle response
	fmt.Printf("Begin to send DelState %+v to %s.\n", request, gc.nodeAddress)
	response, err := clientRPCOutside.DelStateKVService(ctx, request)
	if err != nil {
		fmt.Printf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcGetState(
	request *pb.ClientGetStateKVRequest) (*pb.ClientGetStateKVResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			fmt.Printf("Reconnect TCP to %s fails.\n", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientRPCOutside := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.timeout) * time.Millisecond)
	defer cancel()

	// handle response
	fmt.Printf("Begin to send GetState %+v to %s.\n", request, gc.nodeAddress)
	response, err := clientRPCOutside.GetStateKVService(ctx, request)
	if err != nil {
		fmt.Printf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcAcquireDLock(
	request *pb.ClientAcquireDLockRequest) (*pb.ClientAcquireDLockResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			fmt.Printf("Reconnect TCP to %s fails.\n", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientRPCOutside := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.timeout) * time.Millisecond)
	defer cancel()

	// handle response
	fmt.Printf("Begin to send AcquireDLock %+v to %s.\n", request, gc.nodeAddress)
	response, err := clientRPCOutside.AcquireDLockService(ctx, request)
	if err != nil {
		fmt.Printf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcQueryDLock(
	request *pb.ClientQueryDLockRequest) (*pb.ClientQueryDLockResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			fmt.Printf("Reconnect TCP to %s fails.\n", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientRPCOutside := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.timeout) * time.Millisecond)
	defer cancel()

	// handle response
	fmt.Printf("Begin to send QueryDLock %+v to %s.\n", request, gc.nodeAddress)
	response, err := clientRPCOutside.QueryDLockService(ctx, request)
	if err != nil {
		fmt.Printf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}

func (gc *GrpcCliSrvClientImpl) SendGrpcReleaseDLock(
	request *pb.ClientReleaseDLockRequest) (*pb.ClientReleaseDLockResponse, error) {

	// test whether connection still exists
	if !gc.IsAvailable() {
		// try to reconnect
		err := gc.ReConnect()
		// if reconnect fails
		if err != nil {
			fmt.Printf("Reconnect TCP to %s fails.\n", gc.nodeAddress)
			return nil, err
		}
	}

	// register grpc client
	clientRPCOutside := pb.NewRaftRPCOutsideClientClient(gc.conn)

	// set append entry timeout
	ctx, cancel := context.WithTimeout(context.Background(),
		time.Duration(gc.timeout) * time.Millisecond)
	defer cancel()

	// handle response
	fmt.Printf("Begin to send ReleaseDLock %+v to %s.\n", request, gc.nodeAddress)
	response, err := clientRPCOutside.ReleaseDLockService(ctx, request)
	if err != nil {
		fmt.Printf("Reconnect TCP fails %s.\n", err)
		return nil, err
	}

	return response, nil
}