package client

import (
	"fmt"
	"github.com/dlock_raft/node"
	pb "github.com/dlock_raft/protobuf"
	"github.com/dlock_raft/utils"
	"time"
)

type DLockRaftClientAPI struct {

	// the map from address to
	CliSrvHandler map[string]*node.GrpcCliSrvClientImpl
	// client ID suffix
	ClientIdSuffix string
}

func NewDLockRaftClientAPI(clientIdSuffex string) *DLockRaftClientAPI {
	cliSrcHandler := make(map[string]*node.GrpcCliSrvClientImpl)
	return &DLockRaftClientAPI{CliSrvHandler: cliSrcHandler, ClientIdSuffix: clientIdSuffex}
}

// Warning!!!!!! timeout unit: millisecond
func (drc *DLockRaftClientAPI) preprocessConnections(address string, timeoutOptional ...uint32) {

	// set default timeout
	timeout := uint32(2500)
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

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



func (drc *DLockRaftClientAPI) PutState(address string, key string, value []byte, timeoutOptional ...uint32) bool {

	// set default timeout
	timeout := uint32(2500)
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

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

func (drc *DLockRaftClientAPI) DelState(address string, key string, timeoutOptional ...uint32) bool {

	// set default timeout
	timeout := uint32(2500)
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

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

func (drc *DLockRaftClientAPI) GetState(address string, key string, timeoutOptional ...uint32) []byte {

	// set default timeout
	timeout := uint32(2500)
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

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
		return nil
	} else if err != nil {
		fmt.Printf("Error happens when invoking GetState, %s\n", err)
		return nil
	}

	// committed? redirected? other bugs?
	if response.Success == true {
		fmt.Printf("GetState to %s succeeded, request %+v, response %+v\n", address, request, response)
		return response.Value
	} else {
		fmt.Printf("GetState to %s fails, request %+v\n",
			address, request)
		return nil
	}
}

// expire and timeout should both use ms as unit
func (drc *DLockRaftClientAPI) AcquireDLock(address string, lockName string, expire int64, timeoutOptional ...uint32) bool {

	// set default timeout
	timeout := uint32(2500)
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

	// preprocess connections
	drc.preprocessConnections(address, timeout)
	// construct the request
	// note that expire should have format ns, so we multiply it by 10^6
	request := &pb.ClientAcquireDLockRequest{
		LockName:       lockName,
		ClientIDSuffix: drc.ClientIdSuffix,
		Expire:         expire * 1000000,
		Sequence:       0,
	}

	// start time
	timeStart := time.Now()

	// grpc request
	response, err := drc.CliSrvHandler[address].SendGrpcAcquireDLock(request)
	// error happens in server
	if err != nil {
		fmt.Printf("Error happens when invoking Acquire DLock, %s\n", err)
		return false
	}

	// redirected? succeeded? pending? other bugs?
	startQueryTag := false
	if response.CurrentLeader != ""{
		fmt.Printf("Acquire DLock %s redirected, request %+v, redirected to %s \n",
			address, request, response.CurrentLeader)
		timeExperienced := time.Since(timeStart)
		if time.Duration(int64(timeout)) * time.Millisecond > timeExperienced {
			return drc.AcquireDLock(response.CurrentLeader, lockName, expire,
				timeout - uint32(timeExperienced.Nanoseconds()/1000000))
		} else {
			fmt.Printf("Timeout for acquire DLock, timeout after %s",
				time.Duration(int64(timeout)) * time.Millisecond)
			return false
		}
	} else if response.Pending == false || response.Sequence == 0{
		fmt.Printf("Acquire DLock %s succeeded, request %+v.\n",
			address, request)
		startQueryTag = true
	} else if response.Sequence != 0 {
		fmt.Printf("Acquire DLock %s pending, request %+v, acquirement sequence %d\n",
			address, request, response.Sequence)
		request.Sequence = response.Sequence
	}

	tickerQuery := time.NewTicker(time.Duration(10) * time.Millisecond)
	defer tickerQuery.Stop()
	tickerRefresh := time.NewTicker(time.Duration(250) * time.Millisecond)
	defer tickerRefresh.Stop()
	timer := time.NewTimer(time.Duration(timeout) * time.Millisecond)
	defer timer.Stop()
	requestQuery := &pb.ClientQueryDLockRequest{LockName: lockName}
	_, localIp, err := utils.GetLocalIP()
	if err != nil {
		fmt.Printf("Getting Local ip for Acquire DLock from %s fails, request %+v.\n",
			address, request)
		return false
	}

	for {
		select {
		case <- tickerQuery.C:
			// query whether the
			if startQueryTag {
				responseQuery, err := drc.CliSrvHandler[address].SendGrpcQueryDLock(requestQuery)
				if err != nil {
					fmt.Printf("Error happens when checking state of Acquire DLock, %s\n", err)
					return false
				} else if responseQuery.Owner == utils.FuseClientIp(localIp, drc.ClientIdSuffix){
					fmt.Printf("Acquireing DLock %s confirms success after checking state of Acquire DLock, " +
						"timestamp: %d ms, expire: %d ms\n",
						request.LockName, responseQuery.Timestamp/1000000, responseQuery.Expire/1000000)
					return true
				}
				if responseQuery.PendingNum > 0 {
					fmt.Printf("Acquire DLock %s is still pending, pending acquirement number %d",
						request.LockName, responseQuery.PendingNum)
				} else {
					fmt.Printf("Acquire DLock %s is still pending", request.LockName)
				}
			}
		case <- tickerRefresh.C:
			if !startQueryTag {
				// note that now request.Sequence != 0, meaning an acquirement is pending
				response, err := drc.CliSrvHandler[address].SendGrpcAcquireDLock(request)
				// error happens in server
				// Note that currently we does not support leader redirection in this phrase
				if err != nil || response.CurrentLeader != "" {
					fmt.Printf("Error happens when invoking Acquire DLock, %s\n", err)
					return false
				}
				if response.Pending == false || response.Sequence == 0 {
					fmt.Printf("Acquire DLock %s succeeded, request %+v.\n",
						address, request)
					startQueryTag = true
				}
			}
		case <- timer.C:
			fmt.Printf("Acquire DLock from %s meets server unknown error (timeout), request %+v\n",
				address, request)
			return false
		}
	}
}

