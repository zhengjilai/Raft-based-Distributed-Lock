package node

// the gprc-based server, for client-server transport between client and node

import (
	"errors"
	pb "github.com/dlock_raft/protobuf"
	"github.com/dlock_raft/storage"
	"github.com/dlock_raft/utils"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"net"
	"strings"
	"time"
)

var CliSrvChangeStateInternalError = errors.New("dlock_raft.cli_server: " +
	"something wrong happens inside the server when changing state")
var CliSrvQueryStateInternalError = errors.New("dlock_raft.cli_server: " +
	"something wrong happens inside the server when querying state")
var CliSrvChangeStateTimeoutError = errors.New("dlock_raft.cli_server: " +
	"timeout for the dlock cli-server")
var CliSrvDeadError = errors.New("dlock_raft.gprc_server: " +
	"the node state is Dead, should start it")

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
	gs.NodeRef.NodeLogger.Infof("Begin to precess Put/DelStateKV request, %+v.", request)

	response := &pb.ClientPutStateKVResponse{
		Committed:     false,
		CurrentLeader: "",
	}

	if gs.NodeRef.NodeContextInstance.NodeState != Leader {
		// if node state is not Leader, only return the leader's ip:port
		gs.NodeRef.NodeLogger.Infof("Node %d is not the current leader, begin to redirect the client.",
			gs.NodeRef.NodeConfigInstance.Id.SelfId)
		hopToLeaderId := gs.NodeRef.NodeContextInstance.HopToCurrentLeaderId

		if hopToLeaderId == 0 {
			// no potential leader, then select a random leader
			response.CurrentLeader = utils.RandomObjectInStringList(gs.NodeRef.NodeConfigInstance.Network.PeerAddress)
		} else {
			// have potential leader, then return its ip:port address
			indexToLeaderIpPort := utils.IndexInUint32List(gs.NodeRef.NodeConfigInstance.Id.PeerId, hopToLeaderId)
			response.CurrentLeader = gs.NodeRef.NodeConfigInstance.Network.PeerAddress[indexToLeaderIpPort]
		}
		gs.NodeRef.mutex.Unlock()
		return response, nil
	} else {
		// if node state is Leader, then construct an Entry and trigger an AppendEntries
		// check until the constructed entry is committed
		gs.NodeRef.NodeLogger.Infof("Node %d is the current leader, begin to process the state change",
			gs.NodeRef.NodeConfigInstance.Id.SelfId)

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
		gs.NodeRef.NodeLogger.Infof("Constructed LogEntry index %d term %d " +
			"begin to wait for commitment by raft",
			logEntry.Entry.Index, logEntry.Entry.Term, response)
		// end of construction phrase
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
						gs.NodeRef.NodeLogger.Infof("Constructed LogEntry index %d term %d " +
							"has been committed by raft, now exit with response %+v.",
							fetchedLogEntry.Entry.Index, fetchedLogEntry.Entry.Term, response)
						gs.NodeRef.mutex.Unlock()
						return response, nil
					}
				}
			case <- timer.C:
				// timeout
				gs.NodeRef.NodeLogger.Errorf("Put/DelState Timeout for request %+v, " +
					"constructed LogEntry with index %d, term %d", request, logEntry.Entry.Index, logEntry.Entry.Term)
				return nil, CliSrvChangeStateTimeoutError
			}
		}
	}
}

func (gs *GRPCCliSrvServerImpl) DelStateKVService(ctx context.Context,
	request *pb.ClientDelStateKVRequest) (*pb.ClientDelStateKVResponse, error) {

	// refine the request
	// note that content = nil will trigger a delete state operation in Statemap
	// refer to statemap_memory_impl.go for details
	gs.NodeRef.NodeLogger.Infof("Begin to process DelStateKV request, %+v.", request)
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
		gs.NodeRef.NodeLogger.Infof("End of processing DelStateKV request, %+v.", responseRefine)
		return responseRefine, nil
	}

}

func (gs *GRPCCliSrvServerImpl) GetStateKVService(ctx context.Context,
	request *pb.ClientGetStateKVRequest) (*pb.ClientGetStateKVResponse, error) {

	// lock before doing anything
	gs.NodeRef.mutex.Lock()
	gs.NodeRef.NodeLogger.Infof("Begin to process GetStateKV request, %+v.", request)

	// query KV in state map
	// mote that state in statemap has already been committed
	stateForKey, err := gs.NodeRef.StateMapKVStore.QuerySpecificState(request.Key)
	if err != nil {
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

	gs.NodeRef.NodeLogger.Infof("End of processing GetStateKV request, %+v.", response)
	gs.NodeRef.mutex.Unlock()
	return response, nil

}

// should input the self
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

