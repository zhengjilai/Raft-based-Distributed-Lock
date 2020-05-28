package client

import (
	"fmt"
	"github.com/dlock_raft/node"
	pb "github.com/dlock_raft/protobuf"
	"time"
)

type DLockRaftClientAPI struct {

	// the map from address to
	CliSrvHandler map[string]*node.GrpcCliSrvClientImpl
}

func NewDLockRaftClientAPI() *DLockRaftClientAPI {
	cliSrcHandler := make(map[string]*node.GrpcCliSrvClientImpl)
	return &DLockRaftClientAPI{CliSrvHandler: cliSrcHandler}
}

// Warning!!!!!! timeout unit: millisecond
func (drc *DLockRaftClientAPI) preprocessConnections(address string, timeout uint32) {

	// try to fetch an existing client handler
	cliSrvFetched, ok := drc.CliSrvHandler[address]
	if !ok || cliSrvFetched == nil {
		// if no existing client handler exists, generate a new one
		cliSrvClient := node.NewGrpcCliSrvClientImpl(address, timeout)
		drc.CliSrvHandler[address] = cliSrvClient
		return
	} else {
		cliSrvFetched.SetTimeout(timeout)
		return
	}
}

func (drc *DLockRaftClientAPI) PutState(address string, key string, value []byte, timeout uint32) bool {

	// preprocess connections
	drc.preprocessConnections(address, timeout)
	// construct the request
	request := &pb.ClientPutStateKVRequest{
		Key:     key,
		Content: value,
	}

	// start time
	timeStart := time.Now()

	// grpc request
	response, err := drc.CliSrvHandler[address].SendGrpcPutState(request)
	// error happens in server
	if err == node.CliSrvChangeStateTimeoutError {
		fmt.Printf("PutState to %s meets raft commitment module timeout, request %+v \n",
			address, request)
		return false
	} else if err != nil {
		fmt.Printf("Error happens when invoking PutState, %s\n", err)
		return false
	}

	// committed? redirected? other bugs?
	if response.Committed == true {
		fmt.Printf("PutState to %s succeeded, request %+v\n", address, request)
		return true
	} else if response.Committed == false && response.CurrentLeader != ""{
		fmt.Printf("PutState to %s redirected, request %+v, redirected to %s \n",
			address, request, response.CurrentLeader)
		timeExperienced := time.Since(timeStart)
		if time.Duration(int64(timeout)) * time.Millisecond > timeExperienced {
			return drc.PutState(response.CurrentLeader, key, value,
				timeout - uint32(timeExperienced.Nanoseconds()/1000000))
		} else {
			fmt.Printf("Timeout for client PutState, timeout after %s\n",
				time.Duration(int64(timeout)) * time.Millisecond)
			return false
		}

	} else {
		fmt.Printf("PutState to %s meets server unknown error, request %+v\n",
			address, request)
		return false
	}

}

func (drc *DLockRaftClientAPI) DelState(address string, key string, timeout uint32) bool {

	// preprocess connections
	drc.preprocessConnections(address, timeout)
	// construct the request
	request := &pb.ClientDelStateKVRequest{
		Key:     key,
	}

	// start time
	timeStart := time.Now()

	// grpc request
	response, err := drc.CliSrvHandler[address].SendGrpcDelState(request)
	// error happens in server
	if err == node.CliSrvChangeStateTimeoutError {
		fmt.Printf("DelState to %s meets raft commitment module timeout, request %+v \n",
			address, request)
		return false
	} else if err != nil {
		fmt.Printf("Error happens when invoking DelState, %s\n", err)
		return false
	}

	// committed? redirected? other bugs?
	if response.Committed == true {
		fmt.Printf("DelState to %s succeeded, request %+v\n", address, request)
		return true
	} else if response.Committed == false && response.CurrentLeader != ""{
		fmt.Printf("DelState to %s redirected, request %+v, redirected to %s \n",
			address, request, response.CurrentLeader)
		timeExperienced := time.Since(timeStart)
		if time.Duration(int64(timeout)) * time.Millisecond > timeExperienced {
			return drc.DelState(response.CurrentLeader, key,
				timeout - uint32(timeExperienced.Nanoseconds()/1000000))
		} else {
			fmt.Printf("Timeout for client DelState, timeout after %s",
				time.Duration(int64(timeout)) * time.Millisecond)
			return false
		}

	} else {
		fmt.Printf("DelState to %s meets server unknown error, request %+v\n",
			address, request)
		return false
	}
}

func (drc *DLockRaftClientAPI) GetState(address string, key string, timeout uint32) bool {

	// preprocess connections
	drc.preprocessConnections(address, timeout)
	// construct the request
	request := &pb.ClientGetStateKVRequest{
		Key:     key,
	}

	// grpc request
	response, err := drc.CliSrvHandler[address].SendGrpcGetState(request)
	// error happens in server
	if err == node.CliSrvChangeStateTimeoutError {
		fmt.Printf("GetState to %s meets raft commitment module timeout, request %+v \n",
			address, request)
		return false
	} else if err != nil {
		fmt.Printf("Error happens when invoking GetState, %s\n", err)
		return false
	}

	// committed? redirected? other bugs?
	if response.Success == true {
		fmt.Printf("GetState to %s succeeded, request %+v, response %+v\n", address, request, response)
		return true
	} else {
		fmt.Printf("GetState to %s fails, request %+v\n",
			address, request)
		return false
	}
}