package network
// the gprc-based client, for P2P transport between nodes

import (
	"fmt"
	"github.com/dlock_raft/node"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"time"
)

type GrpcClientImpl struct {

	// the connect server address, format ip:port
	serverAddress string
	conn *grpc.ClientConn
	// the node Object
	node *node.Node
}

func NewGrpcClient(address string, node *node.Node) *GrpcClientImpl {

	// construct the feedback object
	grpcClient := new(GrpcClientImpl)
	grpcClient.serverAddress = address
	grpcClient.conn = nil
	grpcClient.node = node
	return grpcClient
}

func (gc *GrpcClientImpl) StartConnection() error {
	// dial the server without secure option
	conn, err := grpc.Dial(gc.serverAddress, grpc.WithInsecure())
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
		conn, err := grpc.Dial(gc.serverAddress, grpc.WithInsecure())
		if err != nil {
			return err
		}
		gc.conn = conn
	}
	return nil
}

func (gc *GrpcClientImpl) SendDta() {
	go func() {
		for {
			this.ConnState()
		}
	}()

	if !this.IsAvailable() {
		time.Sleep(time.Second)
		fmt.Println("cur state is ", this.conn.GetState())
		this.ReConnect()
		return
	}

	data := <-this.request

	c := pb.NewGreeterClient(this.conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.SayHello(ctx, &pb.HelloRequest{Name: data})
	if err != nil {
		fmt.Printf("could not greet: %v", err)
		this.result <- err.Error()
		return
	}

	this.result <- r.Message

}

func main() {
	var request chan string = make(chan string, 100)
	var result chan string = make(chan string, 100)
	conn := newGrpcConn("127.0.0.1:30051", request, result)

	go func() {
		for {
			conn.SendDta()
		}
	}()

	go func() {
		var index int
		for {
			request <- fmt.Sprintf("%d", index)
			index++
			time.Sleep(time.Second)
		}
	}()

	for {
		v := <-result
		fmt.Println(" result ", v)
	}
}

