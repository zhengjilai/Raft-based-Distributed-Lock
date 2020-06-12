package node

// the gprc-based server, for client-server transport between client and node

import (
	"errors"
	pb "github.com/dlock_raft/protobuf"
	"github.com/dlock_raft/storage"
	"github.com/dlock_raft/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/peer"
	"net"
	"strings"
	"time"
)

var CliSrvChangeStateInternalError = errors.New("dlock_raft.cli_server: " +
	"something wrong happens inside the server when changing state")
var CliSrvQueryStateInternalError = errors.New("dlock_raft.cli_server: " +
	"something wrong happens inside the server when querying state")
var CliSrvChangeStateTimeoutError = errors.New("dlock_raft.cli_server: " +
	"timeout for the dLock cli-server")
var CliSrvDeadError = errors.New("dlock_raft.cli_server: " +
	"the node state is Dead, should start it")
var CliSrvDeleteEmptyKeyValueError = errors.New("dlock_raft.cli_server: " +
	"delState requests an empty key for deletion")
var CliSrvGetEmptyKeyValueError = errors.New("dlock_raft.cli_server: " +
	"getState requests an empty key for getting value")
var CliSrvAcquireDLockInternalError = errors.New("dlock_raft.cli_server: " +
	"something wrong happens inside the server when acquiring dLock")
var CliSrvQueryDLockInternalError = errors.New("dlock_raft.cli_server: " +
	"something wrong happens inside the server when querying dLock info")
var CliSrvReleaseDLockInternalError = errors.New("dlock_raft.cli_server: " +
	"something wrong happens inside the server when releasing dLock")
var CliSrvQueryEmptyDLockError = errors.New("dlock_raft.cli_server: " +
	"QueryDLock requests the information of an non-existing dLock")
var CliSrvGetClientAddressError = errors.New("dlock_raft.cli_server: " +
	"getting client address error")

type GRPCCliSrvServerImpl struct {

	// the actual grpc server object
	cliServer *grpc.Server
	// the NodeRef Object
	NodeRef *Node
}

func NewGRPCCliSrvServerImpl(node *Node) (*GRPCCliSrvServerImpl, error){
	cliServer := new(GRPCCliSrvServerImpl)
	cliServer.NodeRef = node
	return cliServer, nil
}

func (gs *GRPCCliSrvServerImpl) PutStateKVService(ctx context.Context,
	request *pb.ClientPutStateKVRequest) (*pb.ClientPutStateKVResponse, error) {

	// lock before doing anything
	gs.NodeRef.mutex.Lock()

	// if node state is Dead, stop doing anything
	if gs.NodeRef.NodeContextInstance.NodeState == Dead {
		gs.NodeRef.mutex.Unlock()
		return nil, CliSrvDeadError
	}
	gs.NodeRef.NodeLogger.Debugf("Begin to precess Put/DelStateKV request, %+v.", request)

	response := &pb.ClientPutStateKVResponse{
		Committed:     false,
		CurrentLeader: "",
	}

	if gs.NodeRef.NodeContextInstance.NodeState != Leader {
		// if node state is not Leader, only return the leader's ip:port
		gs.NodeRef.NodeLogger.Debugf("Node %d is not the current leader, begin to redirect the client.",
			gs.NodeRef.NodeConfigInstance.Id.SelfId)
		hopToLeaderId := gs.NodeRef.NodeContextInstance.HopToCurrentLeaderId

		if hopToLeaderId == 0 {
			// no potential leader, then select a random leader
			response.CurrentLeader = utils.RandomObjectInStringList(gs.NodeRef.NodeConfigInstance.Network.PeerCliAddress)
		} else {
			// have potential leader, then return its ip:port address
			indexToLeaderIpPort := utils.IndexInUint32List(gs.NodeRef.NodeConfigInstance.Id.PeerId, hopToLeaderId)
			response.CurrentLeader = gs.NodeRef.NodeConfigInstance.Network.PeerCliAddress[indexToLeaderIpPort]
		}
		gs.NodeRef.mutex.Unlock()
		return response, nil
	} else {
		// if node state is Leader, then construct an Entry and trigger an AppendEntries
		// check until the constructed entry is committed
		gs.NodeRef.NodeLogger.Debugf("Node %d is the current leader, begin to process the state change",
			gs.NodeRef.NodeConfigInstance.Id.SelfId)

		// if operation is del state, first check whether a current state is existing
		if request.Content == nil {
			_, err := gs.NodeRef.StateMapKVStore.QuerySpecificState(request.Key)
			if err == storage.InMemoryStateMapKVFetchError {
				gs.NodeRef.NodeLogger.Errorf("DelState requests an empty key: %s.", request.Key)
				gs.NodeRef.mutex.Unlock()
				return response, CliSrvDeleteEmptyKeyValueError
			}
		}

		// construct command, logEntry and insert it into logMemory
		commandKV, err := storage.NewCommandKVStore(request.Key, request.Content)
		if err != nil {
			gs.NodeRef.NodeLogger.Errorf("Put/DelStateKV Error: %s.", err)
			gs.NodeRef.mutex.Unlock()
			return nil, CliSrvChangeStateInternalError
		}
		logEntry, err := storage.NewLogEntry(gs.NodeRef.NodeContextInstance.CurrentTerm,
			gs.NodeRef.LogEntryInMemory.MaximumIndex() + 1, commandKV)
		if err != nil {
			gs.NodeRef.NodeLogger.Errorf("Put/DelStateKV Error: %s.", err)
			gs.NodeRef.mutex.Unlock()
			return nil, CliSrvChangeStateInternalError
		}
		err = gs.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
		if err != nil {
			gs.NodeRef.NodeLogger.Errorf("Put/DelStateKV Error: %s.", err)
			gs.NodeRef.mutex.Unlock()
			return nil, CliSrvChangeStateInternalError
		}
		gs.NodeRef.NodeLogger.Debugf("Constructed LogEntry index %d term %d " +
			"begin to wait for commitment by raft",
			logEntry.Entry.Index, logEntry.Entry.Term)
		// end of construction phrase, trigger append entry
		gs.NodeRef.NodeContextInstance.TriggerAEChannel()
		gs.NodeRef.mutex.Unlock()

		// the ticker for check
		ticker := time.NewTicker(time.Duration(gs.NodeRef.NodeConfigInstance.Parameters.PollingInterval) * time.Millisecond)
		defer ticker.Stop()
		timer := time.NewTimer(time.Duration(gs.NodeRef.NodeConfigInstance.Parameters.StateChangeTimeout) * time.Millisecond)
		defer timer.Stop()

		for {
			select {
			case <- ticker.C:
				// routine check
				gs.NodeRef.mutex.Lock()
				// check the logMemory, whether the LogEntry has been committed by raft
				if gs.NodeRef.NodeContextInstance.CommitIndex >= logEntry.Entry.Index {
					// confirm the logEntry has been committed
					fetchedLogEntry, err := gs.NodeRef.LogEntryInMemory.FetchLogEntry(logEntry.Entry.Index)
					if err != nil {
						gs.NodeRef.NodeLogger.Errorf("Put/DelStateKV Error: %s.", err)
						gs.NodeRef.mutex.Unlock()
						return nil, CliSrvChangeStateInternalError
					}
					// check if the term is right
					if fetchedLogEntry.Entry.Term == logEntry.Entry.Term{
						response.Committed = true
						gs.NodeRef.NodeLogger.Debugf("Constructed LogEntry index %d term %d " +
							"has been committed by raft, now exit with response %+v.",
							fetchedLogEntry.Entry.Index, fetchedLogEntry.Entry.Term, response)
						gs.NodeRef.mutex.Unlock()
						return response, nil
					}
				}
				// don't forget to unlock
				gs.NodeRef.mutex.Unlock()

			case <- timer.C:
				// timeout
				gs.NodeRef.NodeLogger.Errorf("Put/DelState Timeout for request %+v, " +
					"constructed LogEntry with index %d, term %d", request, logEntry.Entry.Index, logEntry.Entry.Term)
				return response, CliSrvChangeStateTimeoutError
			}
		}
	}
}

func (gs *GRPCCliSrvServerImpl) DelStateKVService(ctx context.Context,
	request *pb.ClientDelStateKVRequest) (*pb.ClientDelStateKVResponse, error) {

	// refine the request
	// note that content = nil will trigger a delete state operation in Statemap
	// refer to statemap_memory_impl.go for details
	gs.NodeRef.NodeLogger.Debugf("Begin to process DelStateKV request, %+v.", request)
	requestRefine := &pb.ClientPutStateKVRequest{
		Key:     request.Key,
		Content: nil,
	}
	response, err := gs.PutStateKVService(ctx, requestRefine)
	if err != nil {
		return nil, err
	} else {
		// refine the response
		responseRefine := &pb.ClientDelStateKVResponse{
			Committed:     response.Committed,
			CurrentLeader: response.CurrentLeader,
		}
		gs.NodeRef.NodeLogger.Debugf("End of processing DelStateKV request, %+v.", responseRefine)
		return responseRefine, nil
	}

}

func (gs *GRPCCliSrvServerImpl) GetStateKVService(ctx context.Context,
	request *pb.ClientGetStateKVRequest) (*pb.ClientGetStateKVResponse, error) {

	// lock before doing anything
	gs.NodeRef.mutex.Lock()
	gs.NodeRef.NodeLogger.Debugf("Begin to process GetStateKV request, %+v.", request)

	// query KV in state map
	// mote that state in statemap has already been committed
	stateForKey, err := gs.NodeRef.StateMapKVStore.QuerySpecificState(request.Key)
	if err == storage.InMemoryStateMapKVFetchError {
		gs.NodeRef.NodeLogger.Errorf("GetState requests an empty key: %s.", request.Key)
		gs.NodeRef.mutex.Unlock()
		return nil, CliSrvGetEmptyKeyValueError
	} else if err != nil {
		gs.NodeRef.NodeLogger.Errorf("GetStateKV Error: %s.", err)
		gs.NodeRef.mutex.Unlock()
		return nil, CliSrvQueryStateInternalError
	}
	stateForKeyByte, ok := stateForKey.([]byte)
	if !ok {
		gs.NodeRef.NodeLogger.Error("GetStateKV Error: the fetched value cannot be deserialized.")
		gs.NodeRef.mutex.Unlock()
		return nil, CliSrvQueryStateInternalError
	}
	// construct response if no bug happened
	response := &pb.ClientGetStateKVResponse{
		Success: true,
		Value:   stateForKeyByte,
	}

	gs.NodeRef.NodeLogger.Debugf("End of processing GetStateKV request, %+v.", response)
	gs.NodeRef.mutex.Unlock()
	return response, nil

}

func (gs *GRPCCliSrvServerImpl) AcquireDLockService(ctx context.Context,
	request *pb.ClientAcquireDLockRequest) (*pb.ClientAcquireDLockResponse, error) {

	gs.NodeRef.NodeLogger.Debugf("Begin to process AcquireDLock request, %+v.", request)

	// get client ip address (used to construct clientId)
	ip, err := gs.GetIPAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}

	response := &pb.ClientAcquireDLockResponse{
		Pending:       true,
		Sequence:      0,
		CurrentLeader: "",
	}

	// lock before doing anything
	gs.NodeRef.mutex.Lock()
	defer gs.NodeRef.mutex.Unlock()

	if gs.NodeRef.NodeContextInstance.NodeState != Leader {
		hopToLeaderId := gs.NodeRef.NodeContextInstance.HopToCurrentLeaderId
		if hopToLeaderId == 0 {
			// no potential leader, then select a random leader
			response.CurrentLeader = utils.RandomObjectInStringList(gs.NodeRef.NodeConfigInstance.Network.PeerCliAddress)
		} else {
			// have potential leader, then return its ip:port address
			indexToLeaderIpPort := utils.IndexInUint32List(gs.NodeRef.NodeConfigInstance.Id.PeerId, hopToLeaderId)
			response.CurrentLeader = gs.NodeRef.NodeConfigInstance.Network.PeerCliAddress[indexToLeaderIpPort]
		}
		gs.NodeRef.NodeLogger.Debugf("Acquire DLock request terminates as current node is not Leader, %+v.", response)
		return response, nil

	} else {

		clientId := utils.FuseClientIp(ip, request.ClientIDSuffix)
		timestamp := time.Now().UnixNano()

		if request.Sequence != 0 {
			// request.sequence != 0 means the client wants to refresh a dLock acquirement by its sequence
			refreshSuccess, err := gs.NodeRef.DlockInterchangeInstance.RefreshAcquirementBySequence(request.LockName, request.Sequence, timestamp)
			if err != nil {
				gs.NodeRef.NodeLogger.Debugf("Refresh acquirement dLock %s fails, error %s",
					request.LockName, err)
				return nil, CliSrvAcquireDLockInternalError
			} else {
				response.Pending = refreshSuccess
				if refreshSuccess {
					response.Sequence = request.Sequence
					gs.NodeRef.NodeLogger.Debugf("Refresh acquirement dLock %s by sequence %d succeeded, response %+v",
						request.LockName, request.Sequence, response)
				} else {
					gs.NodeRef.NodeLogger.Debugf("Refresh acquirement dLock %s by sequence %d " +
						"failed, since the acquirement does not exist or has already expired, response %+v",
						request.LockName, request.Sequence, response)
				}
				return response, nil
			}
		} else {
			// now request.sequence == 0, meaning client wants to acquire a new dLock
			// construct command, lockNonce and timestamp will be refreshed in AcquireDLock, don't worry (smile)
			command, err := storage.NewCommandDLock(0, request.LockName, clientId, timestamp, request.Expire)
			if err != nil {
				return nil, CliSrvAcquireDLockInternalError
			}
			sequence, err := gs.NodeRef.DlockInterchangeInstance.AcquireDLock(request.LockName, timestamp, command)
			if err != nil {
				gs.NodeRef.NodeLogger.Debugf("Acquire dLock %s fails due to %s", request.LockName, err)
				return nil, CliSrvAcquireDLockInternalError
			}
			if sequence == 0 {
				// meaning that a LogEntry is directly appended to LogMemory
				response.Pending = false
				gs.NodeRef.NodeLogger.Debugf("Trigger Acquire dLock %s succeeded " +
					"by directly appending a LogEntry, response %+v", request.LockName, response)
				return response, nil
			} else {
				response.Pending = true
				response.Sequence = sequence
				gs.NodeRef.NodeLogger.Debugf("Trigger Acquire dLock %s by inserting an acquirement to pending list," +
					" sequence %d, response %+v", request.LockName, sequence, response)
				return response, nil
			}
		}
	}
}

func (gs *GRPCCliSrvServerImpl) QueryDLockService(ctx context.Context,
	request *pb.ClientQueryDLockRequest) (*pb.ClientQueryDLockResponse, error) {
	// lock before doing anything
	gs.NodeRef.mutex.Lock()
	defer gs.NodeRef.mutex.Unlock()
	gs.NodeRef.NodeLogger.Debugf("Begin to process QueryDLock request, %+v.", request)

	// construct response
	response := &pb.ClientQueryDLockResponse{
		Owner:      "",
		Nonce:      0,
		Timestamp:  0,
		Expire:     0,
		PendingNum: -1,
	}

	// update response from statemap state
	state, err := gs.NodeRef.StateMapDLock.QuerySpecificState(request.LockName)
	if err == storage.InMemoryStateMapDLockFetchError {
		return nil, CliSrvQueryEmptyDLockError
	} else if err != nil {
		return nil, CliSrvQueryDLockInternalError
	}

	stateDecoded, ok := state.(*storage.DlockState)
	if !ok {
		return nil, CliSrvQueryDLockInternalError
	}
	response.Owner = stateDecoded.Owner
	response.Timestamp = stateDecoded.Timestamp
	response.Expire = stateDecoded.Expire
	response.Nonce = stateDecoded.LockNonce

	// query acquirement info only if the node state is leader
	if gs.NodeRef.NodeContextInstance.NodeState == Leader {
		response.PendingNum = gs.NodeRef.DlockInterchangeInstance.QueryDLockAcquirementInfo(request.LockName)
	}

	gs.NodeRef.NodeLogger.Debugf("End of processing QueryDLock request, %+v.", response)
	return response, nil
}

func (gs *GRPCCliSrvServerImpl) ReleaseDLockService(ctx context.Context,
	request *pb.ClientReleaseDLockRequest) (*pb.ClientReleaseDLockResponse, error) {

	gs.NodeRef.NodeLogger.Debugf("Begin to process ReleaseDLock request, %+v.", request)

	// get client ip address (used to construct clientId)
	ip, err := gs.GetIPAddressFromContext(ctx)
	if err != nil {
		return nil, err
	}
	response := &pb.ClientReleaseDLockResponse{
		Released:      false,
		CurrentLeader: "",
	}

	// lock before doing anything
	gs.NodeRef.mutex.Lock()

	if gs.NodeRef.NodeContextInstance.NodeState != Leader {
		hopToLeaderId := gs.NodeRef.NodeContextInstance.HopToCurrentLeaderId
		if hopToLeaderId == 0 {
			// no potential leader, then select a random leader
			response.CurrentLeader = utils.RandomObjectInStringList(gs.NodeRef.NodeConfigInstance.Network.PeerCliAddress)
		} else {
			// have potential leader, then return its ip:port address
			indexToLeaderIpPort := utils.IndexInUint32List(gs.NodeRef.NodeConfigInstance.Id.PeerId, hopToLeaderId)
			response.CurrentLeader = gs.NodeRef.NodeConfigInstance.Network.PeerCliAddress[indexToLeaderIpPort]
		}
		gs.NodeRef.NodeLogger.Debugf("Release DLock request terminates as current node is not Leader, %+v.", response)
		gs.NodeRef.mutex.Unlock()
		return response, nil
	} else {
		clientId := utils.FuseClientIp(ip, request.ClientIDSuffix)
		timestamp := time.Now().UnixNano()
		releaseInfo, err := gs.NodeRef.DlockInterchangeInstance.ReleaseDLock(request.LockName, clientId, timestamp)
		if err != nil || releaseInfo == ErrorReserve{
			return nil, CliSrvReleaseDLockInternalError
		}
		if releaseInfo == NoDLockExist || releaseInfo == AlreadyReleased || releaseInfo == TriggerReleaseSuccess {
			response.Released = true
			gs.NodeRef.NodeLogger.Debugf("End of processing ReleaseDLock request (%d), %+v.", releaseInfo, response)
			gs.NodeRef.mutex.Unlock()
			return response, nil
		} else {
			// now releaseInfo only has one potential: TriggerReleaseLater
			// so we begin to trigger release periodically, until timeout or release succeeded

			// the ticker for check
			ticker := time.NewTicker(time.Duration(gs.NodeRef.NodeConfigInstance.Parameters.PollingInterval) * time.Millisecond)
			defer ticker.Stop()
			timer := time.NewTimer(time.Duration(gs.NodeRef.NodeConfigInstance.Parameters.StateChangeTimeout) * time.Millisecond)
			defer timer.Stop()
			gs.NodeRef.mutex.Unlock()

			for {
				select {
				case <-ticker.C:
					gs.NodeRef.mutex.Lock()
					timestamp := time.Now().UnixNano()
					releaseInfo, err := gs.NodeRef.DlockInterchangeInstance.ReleaseDLock(
						request.LockName, clientId, timestamp)
					if err != nil || releaseInfo == ErrorReserve {
						gs.NodeRef.mutex.Unlock()
						return nil, CliSrvReleaseDLockInternalError
					}
					if releaseInfo == NoDLockExist || releaseInfo == AlreadyReleased || releaseInfo == TriggerReleaseSuccess {
						response.Released = true
						gs.NodeRef.NodeLogger.Debugf("End of processing ReleaseDLock request (%d), %+v.",
							releaseInfo, response)
						gs.NodeRef.mutex.Unlock()
						return response, nil
					}
					gs.NodeRef.mutex.Unlock()
				case <-timer.C:
					return response, nil
				}
			}
		}
	}
}

// get ip address (string) from context
// ip address is often used to get ClientId (owner)
func (gs *GRPCCliSrvServerImpl) GetIPAddressFromContext(ctx context.Context) (string, error){
	peerFromContext, ok := peer.FromContext(ctx)
	if !ok || peerFromContext.Addr == nil {
		return "", CliSrvGetClientAddressError
	}
	ip, _, err := net.SplitHostPort(peerFromContext.Addr.String())
	if err != nil {
		return "", CliSrvGetClientAddressError
	}
	return ip, nil
}

// start cli-srv grpc service
func (gs *GRPCCliSrvServerImpl) StartService() {

	// get the listing address
	address := gs.NodeRef.NodeConfigInstance.Network.SelfCliAddress
	splittedAddress := strings.Split(address, ":")
	if len(splittedAddress) != 2 {
		gs.NodeRef.NodeLogger.Errorf("GRPC address: %s", GrpcP2PServerAddressError)
		return
	}

	// listening at a specific port
	listener, err := net.Listen("tcp", ":" + splittedAddress[1])
	if err != nil {
		gs.NodeRef.NodeLogger.Errorf("GRPC Listener init error: %s", err)
		return
	}

	// start a new grpc server
	gs.cliServer = grpc.NewServer()
	pb.RegisterRaftRPCOutsideClientServer(gs.cliServer, gs)

	// start service
	err = gs.cliServer.Serve(listener)
	if err != nil {
		gs.NodeRef.NodeLogger.Errorf("GRPC server init error: %s", err)
		return
	}
}

