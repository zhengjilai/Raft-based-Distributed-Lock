package client

import
(
	"fmt"
	"github.com/dlock_raft/node"
	pb "github.com/dlock_raft/protobuf"
	"github.com/dlock_raft/utils"
	"time"
)

type DLockRaftClientAPI struct {

	// the map from address to
	CliSrvHandler map[string]*node.GrpcCliSrvClientImpl
	// client ID, UUID, generated randomly
	ClientId string
}

func NewDLockRaftClientAPI(clientIdPreset ...string) *DLockRaftClientAPI {

	// set default clientId, or use preset one
	var clientId string
	if len(clientIdPreset) > 0 && len(clientIdPreset[0]) != 27 {
		clientId = clientIdPreset[0]
		fmt.Printf("Preset Client Id (UUID) %s succeeded.\n", clientId)
	} else {
		clientId = utils.GenKsuid()
		fmt.Printf("Generate random Client Id (UUID) %s succeeded.\n", clientId)
	}
	cliSrcHandler := make(map[string]*node.GrpcCliSrvClientImpl)
	return &DLockRaftClientAPI{
		CliSrvHandler: cliSrcHandler,
		ClientId: clientId,
	}
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
		fmt.Printf("PutState to %s meets raft commitment module timeout, request %+v.\n",
			address, request)
		return false
	} else if err != nil {
		fmt.Printf("Error happens when invoking PutState, %s.\n", err)
		return false
	}

	// committed? redirected? other bugs?
	if response.Committed == true {
		fmt.Printf("PutState to %s succeeded, request %+v.\n", address, request)
		return true
	} else if response.Committed == false && response.CurrentLeader != ""{
		fmt.Printf("PutState to %s redirected, request %+v, redirected to %s.\n",
			address, request, response.CurrentLeader)
		timeExperienced := time.Since(timeStart)
		if time.Duration(int64(timeout)) * time.Millisecond > timeExperienced {
			return drc.PutState(response.CurrentLeader, key, value,
				timeout - uint32(timeExperienced.Nanoseconds()/1000000))
		} else {
			fmt.Printf("Timeout for client PutState, timeout after %s.\n",
				time.Duration(int64(timeout)) * time.Millisecond)
			return false
		}

	} else {
		fmt.Printf("PutState to %s meets server unknown error, request %+v.\n",
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
		fmt.Printf("DelState to %s meets raft commitment module timeout, request %+v.\n",
			address, request)
		return false
	} else if err != nil {
		fmt.Printf("Error happens when invoking DelState, %s.\n", err)
		return false
	}

	// committed? redirected? other bugs?
	if response.Committed == true {
		fmt.Printf("DelState to %s succeeded, request %+v.\n", address, request)
		return true
	} else if response.Committed == false && response.CurrentLeader != ""{
		fmt.Printf("DelState to %s redirected, request %+v, redirected to %s.\n",
			address, request, response.CurrentLeader)
		timeExperienced := time.Since(timeStart)
		if time.Duration(int64(timeout)) * time.Millisecond > timeExperienced {
			return drc.DelState(response.CurrentLeader, key,
				timeout - uint32(timeExperienced.Nanoseconds()/1000000))
		} else {
			fmt.Printf("Timeout for client DelState, timeout after %s.\n",
				time.Duration(int64(timeout)) * time.Millisecond)
			return false
		}

	} else {
		fmt.Printf("DelState to %s meets server unknown error, request %+v.\n",
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
		fmt.Printf("GetState to %s meets raft commitment module timeout, request %+v.\n",
			address, request)
		return nil
	} else if err != nil {
		fmt.Printf("Error happens when invoking GetState, %s.\n", err)
		return nil
	}

	// committed? redirected? other bugs?
	if response.Success == true {
		fmt.Printf("GetState to %s succeeded, request %+v, response %+v.\n", address, request, response)
		return response.Value
	} else {
		fmt.Printf("GetState to %s fails, request %+v.\n",
			address, request)
		return nil
	}
}

// API for Acquire DLock, may block before timeout if the lock is not obtained
// expire and timeout should both use ms as unit
func (drc *DLockRaftClientAPI) AcquireDLock(address string,
	lockName string, expire int64, timeoutOptional ...uint32) bool {

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
		ClientID: 		drc.ClientId,
		Expire:         expire * 1000000,
		Sequence:       0,
	}

	// start time
	timeStart := time.Now()

	// grpc request
	response, err := drc.CliSrvHandler[address].SendGrpcAcquireDLock(request)
	// error happens in server
	if err != nil {
		fmt.Printf("Error happens when invoking Acquire DLock, %s.\n", err)
		return false
	}

	// redirected? succeeded? pending? other bugs?
	startQueryTag := false
	if response.CurrentLeader != ""{
		fmt.Printf("Acquire DLock %s redirected, request %+v, redirected to %s.\n",
			address, request, response.CurrentLeader)
		timeExperienced := time.Since(timeStart)
		if time.Duration(int64(timeout)) * time.Millisecond > timeExperienced {
			return drc.AcquireDLock(response.CurrentLeader, lockName, expire,
				timeout - uint32(timeExperienced.Nanoseconds()/1000000))
		} else {
			fmt.Printf("Timeout for acquire DLock, timeout after %s.\n",
				time.Duration(int64(timeout)) * time.Millisecond)
			return false
		}
	} else if response.Pending == false || response.Sequence == 0 {
		fmt.Printf("Acquire DLock %s succeeded, request %+v.\n",
			address, request)
		startQueryTag = true
	} else if response.Sequence != 0 {
		fmt.Printf("Acquire DLock %s pending, request %+v, acquirement sequence %d.\n",
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
	if err != nil {
		fmt.Printf("Getting Local ip for Acquire DLock from %s fails, request %+v.\n",
			address, request)
		return false
	}

	for {
		select {
		case <- tickerQuery.C:
			// judge whether the query process has started up
			if startQueryTag {
				responseQuery, err := drc.CliSrvHandler[address].SendGrpcQueryDLock(requestQuery)
				if err != nil {
					fmt.Printf("Error happens when checking state of Acquire DLock, %s.\n", err)
					return false
				} else if responseQuery.Owner == drc.ClientId{
					fmt.Printf("Acquireing DLock %s confirms success after checking state of Acquire DLock, " +
						"timestamp: %d ms, expire: %d ms\n",
						request.LockName, responseQuery.Timestamp/1000000, responseQuery.Expire/1000000)
					return true
				}
				if responseQuery.PendingNum > 0 {
					fmt.Printf("Acquire DLock %s is still pending, pending acquirement number %d.\n",
						request.LockName, responseQuery.PendingNum)
				} else {
					fmt.Printf("Acquire DLock %s is still pending.\n", request.LockName)
				}
			}
		case <- tickerRefresh.C:
			if !startQueryTag {
				// note that now request.Sequence != 0, meaning an acquirement is pending
				response, err := drc.CliSrvHandler[address].SendGrpcAcquireDLock(request)
				// error happens in server
				// Note that currently we does not support leader redirection in this phrase
				if err != nil || response.CurrentLeader != "" {
					fmt.Printf("Error happens when invoking Acquire DLock, %s.\n", err)
					return false
				}
				if response.Pending == false || response.Sequence == 0 {
					fmt.Printf("Acquire DLock %s succeeded, request %+v.\n",
						address, request)
					startQueryTag = true
				}
			}
		case <- timer.C:
			fmt.Printf("Acquire DLock from %s meets server unknown error (timeout), request %+v.\n",
				address, request)
			return false
		}
	}
}

// DLockQueryInfo is used for indicating the basic info of an DLock
// Note that it is written basically for decoupling protobuf response with client-side objects
type DLockQueryInfo struct {
	// current owner, "" for nobody (not acquired yet)
	Owner string
	// dlock nonce
	Nonce uint32
	// the timestamp when dlock last refreshed, format: ns
	Timestamp int64
	// the current dlock expire, format: ns
	Expire int64
	// the pending acquirement number
	// return a non-negative number only when client ask the leader
	PendingNum int32
}

// expire and timeout should both use ms as unit
// API for Query DLock, never block if the lock is not obtained
func (drc *DLockRaftClientAPI) QueryDLock(
	address string, lockName string, timeoutOptional ...uint32) (*DLockQueryInfo, bool) {

	// set default timeout
	timeout := uint32(2500)
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

	// preprocess connections
	drc.preprocessConnections(address, timeout)
	// construct the request
	request := &pb.ClientQueryDLockRequest{
		LockName: lockName,
	}

	// grpc request
	response, err := drc.CliSrvHandler[address].SendGrpcQueryDLock(request)
	// error happens in server
	if err != nil {
		fmt.Printf("Error happens when invoking Query DLock, %s.\n", err)
		return nil, false
	}
	responseLocal := &DLockQueryInfo{
		Owner: response.Owner,
		Nonce: response.Nonce,
		Timestamp: response.Timestamp,
		Expire: response.Expire,
		PendingNum: response.PendingNum,
	}
	return responseLocal, true
}

// API for Release DLock, never block if something delay happened at server side
func (drc *DLockRaftClientAPI) ReleaseDLock(
	address string, lockName string, timeoutOptional ...uint32) bool {

	// set default timeout
	timeout := uint32(2500)
	if len(timeoutOptional) > 0 {
		timeout = timeoutOptional[0]
	}

	// preprocess connections
	drc.preprocessConnections(address, timeout)
	// construct the request
	// note that expire should have format ns, so we multiply it by 10^6
	request := &pb.ClientReleaseDLockRequest{
		LockName:       lockName,
		ClientID: 		drc.ClientId,
	}

	// start time
	timeStart := time.Now()
	// grpc request
	response, err := drc.CliSrvHandler[address].SendGrpcReleaseDLock(request)
	// error happens in server
	if err != nil {
		fmt.Printf("Error happens when invoking Acquire DLock, %s\n", err)
		return false
	}

	// redirected? succeeded? pending? other bugs?
	if response.CurrentLeader != ""{
		fmt.Printf("Acquire DLock %s redirected, request %+v, redirected to %s.\n",
			address, request, response.CurrentLeader)
		timeExperienced := time.Since(timeStart)
		if time.Duration(int64(timeout)) * time.Millisecond > timeExperienced {
			return drc.ReleaseDLock(response.CurrentLeader, lockName,
				timeout - uint32(timeExperienced.Nanoseconds()/1000000))
		} else {
			fmt.Printf("Timeout for acquire DLock, timeout after %s",
				time.Duration(int64(timeout)) * time.Millisecond)
			return false
		}
	} else if response.Released == true {
		fmt.Printf("DLock %s has been released, meaning you does not possess it now.\n", lockName)
		return true
	} else {
		fmt.Printf("Releasing DLock %s failed, meaning nothing have been done for releasing dlock.\n", lockName)
		return false
	}
}