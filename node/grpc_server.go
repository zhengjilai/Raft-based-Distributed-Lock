package node

// the gprc-based server, for P2P transport between nodes

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

var GRPCServerAddressError = errors.New("dlock_raft.gprc_server: " +
	"the listening address is wrong, should have format ip:port")
var GRPCServerDeadError = errors.New("dlock_raft.gprc_server: " +
	"the node state is Dead, should start it")

type GrpcServerImpl struct {

	// the actual grpc server object
	grpcServer *grpc.Server
	// the NodeRef Object
	NodeRef *Node
}

func NewGrpcServer(node *Node) (*GrpcServerImpl, error){
	grpcServer := new(GrpcServerImpl)
	grpcServer.NodeRef = node
	return grpcServer, nil
}

func (gs *GrpcServerImpl) AppendEntriesService(ctx context.Context,
	request *pb.AppendEntriesRequest) (*pb.AppendEntriesResponse, error) {

	// lock before doing anything
	gs.NodeRef.mutex.Lock()
	defer gs.NodeRef.mutex.Unlock()

	// if node state is Dead, stop doing anything
	if gs.NodeRef.NodeContextInstance.NodeState == Dead {
		return nil, GRPCServerDeadError
	}
	gs.NodeRef.NodeLogger.Infof("Begin to precess AppendEntries request, %+v.", request)

	response := &pb.AppendEntriesResponse{
		Term:             gs.NodeRef.NodeContextInstance.CurrentTerm,
		NodeId:           gs.NodeRef.NodeConfigInstance.Id.SelfId,
		ConflictEntryIndex: 0,
		ConflictEntryTerm: 0,
		CommitEntryIndex: gs.NodeRef.NodeContextInstance.CommitIndex,
		// whether the AppendEntry prevIndex/Term matches the local LogMemory
		Success: false,
	}

	// the remote term exceeds local term, should become follower
	if request.Term > gs.NodeRef.NodeContextInstance.CurrentTerm {
		gs.NodeRef.NodeLogger.Infof("AppendEntry term %d is greater than current term %d.",
			request.Term, gs.NodeRef.NodeContextInstance.CurrentTerm)
		gs.NodeRef.BecomeFollower(request.Term)
		// refresh the current term
		gs.NodeRef.NodeContextInstance.CurrentTerm = request.Term
		// reset the election start time, for receiving heart beat (AppendEntry)
		gs.NodeRef.NodeContextInstance.electionRestartTime = time.Now()
	}
	// if it is exactly the same term, then an AppendEntry should always come from the leader
	if request.Term == gs.NodeRef.NodeContextInstance.CurrentTerm {
		// become follower when hear from the current leader
		if gs.NodeRef.NodeContextInstance.NodeState != Follower{
			gs.NodeRef.BecomeFollower(request.Term)
		}
		// reset the election start time, for receiving heart beat (AppendEntry)
		gs.NodeRef.NodeContextInstance.electionRestartTime = time.Now()
		// judge if the term exists in LogMemory, or it is the first AppendEntry
		if request.PrevEntryIndex <= gs.NodeRef.LogEntryInMemory.MaximumIndex() {
			// prevIndex is a solid entry
			if request.PrevEntryIndex != 0{
				entryForPrevIndex, err := gs.NodeRef.LogEntryInMemory.FetchLogEntry(request.PrevEntryIndex)
				if err != nil {
					return nil, err
				}
				// meaning the entry in LogMemory does match prev of leader
				if entryForPrevIndex.Entry.Term == request.PrevEntryTerm {
					response.Success = true
				}
				// prevIndex = 0, an extreme situation
			} else {
				response.Success = true
			}
		}

		if response.Success {
			// if prev index matches or prevIndex = 0, then insert the whole entry list
			logEntryList := make([]*storage.LogEntry, len(request.EntryList))
			for i, entry := range request.EntryList{
				logEntry, err := storage.NewLogEntryList(entry)
				if err != nil {
					return nil, err
				}
				logEntryList[i] = logEntry
			}
			err := gs.NodeRef.LogEntryInMemory.InsertValidEntryList(logEntryList)
			if err != nil {
				return nil, err
			}
			// only if success, then try to commit the some index
			if request.CommitEntryIndex > gs.NodeRef.NodeContextInstance.CommitIndex {
				gs.NodeRef.NodeContextInstance.CommitIndex = utils.Uint64Min(request.CommitEntryIndex,
					gs.NodeRef.LogEntryInMemory.MaximumIndex())
				// trigger the commit goroutine
				gs.NodeRef.NodeContextInstance.CommitChan <- struct{}{}
				response.CommitEntryIndex = gs.NodeRef.NodeContextInstance.CommitIndex
			}

		} else {
			// if prev index does not match, find the conflict index and term for leader
			maximumIndex := gs.NodeRef.LogEntryInMemory.MaximumIndex()
			if request.PrevEntryIndex > maximumIndex {
				// > maximum index, then use maximum index in LogMemory
				entryForMaxIndex, err := gs.NodeRef.LogEntryInMemory.FetchLogEntry(maximumIndex)
				if err != nil {
					return nil, err
				}
				response.ConflictEntryIndex = maximumIndex
				response.ConflictEntryTerm = entryForMaxIndex.Entry.Term
			} else if request.PrevEntryIndex != 0{
				// else if !=0, then use the prevIndex in LogMemory
				entryForPrevIndex, err := gs.NodeRef.LogEntryInMemory.FetchLogEntry(request.PrevEntryIndex)
				if err != nil {
					return nil, err
				}
				response.ConflictEntryIndex = request.PrevEntryIndex
				response.ConflictEntryTerm = entryForPrevIndex.Entry.Term
			}
		}
	}
	// if current term > request term, then tell the leader that it is obsolete
	response.Term = gs.NodeRef.NodeContextInstance.CurrentTerm
	gs.NodeRef.NodeLogger.Infof("The AppendEntry response is %+v", response)
	return response, nil
}

func (gs *GrpcServerImpl) CandidateVotesService(ctx context.Context,
	request *pb.CandidateVotesRequest) (*pb.CandidateVotesResponse, error) {

	// lock before doing anything
	gs.NodeRef.mutex.Lock()
	defer gs.NodeRef.mutex.Unlock()

	// if node state is Dead, stop doing anything
	if gs.NodeRef.NodeContextInstance.NodeState == Dead {
		return nil, GRPCServerDeadError
	}
	gs.NodeRef.NodeLogger.Infof("Begin to precess Candidate Votes request, %+v.", request)

	// the original response
	response := &pb.CandidateVotesResponse{
		Term:     gs.NodeRef.NodeContextInstance.CurrentTerm,
		NodeId:   gs.NodeRef.NodeConfigInstance.Id.SelfId,
		Accepted: false,
	}

	// request term exceeds the local term, then become a follower
	if request.Term > gs.NodeRef.NodeContextInstance.CurrentTerm {
		gs.NodeRef.NodeLogger.Infof("Candidate Vote term %d is greater than current term %d.",
			request.Term, gs.NodeRef.NodeContextInstance.CurrentTerm)
		gs.NodeRef.BecomeFollower(request.Term)
		// refresh the current term
		gs.NodeRef.NodeContextInstance.CurrentTerm = request.Term
		// reset the election module, for receiving Candidate vote from higher term
		gs.NodeRef.NodeContextInstance.electionRestartTime = time.Now()

	} else if (request.Term == gs.NodeRef.NodeContextInstance.CurrentTerm) &&
		(gs.NodeRef.NodeContextInstance.VotedPeer == 0 ||
			gs.NodeRef.NodeContextInstance.VotedPeer == request.NodeId) {
		// term number equals, and node has not voted for any peer except the remote candidate
		// note that votedPeer = 0 means the local node haz voted for no one

		// get the local maximum index and corresponding term
		maximumEntry, err := gs.NodeRef.LogEntryInMemory.FetchLogEntry(gs.NodeRef.LogEntryInMemory.MaximumIndex())
		if err != nil {
			return nil, err
		}
		// the node will only vote for those candidate with:
		// 1. has longer LogEntry list and has the same term
		// 2. has a LogEntryList with higher term
		if request.PrevEntryTerm > maximumEntry.Entry.Term ||
			(request.PrevEntryTerm == maximumEntry.Entry.Term &&
				request.PrevEntryIndex > maximumEntry.Entry.Term ) {
			// vote for the specific candidate
			response.Accepted = true
			gs.NodeRef.NodeContextInstance.VotedPeer = request.NodeId
			// reset the election module, for receiving Candidate vote from the same term, and voting for it
			gs.NodeRef.NodeContextInstance.electionRestartTime = time.Now()
		}
	}

	// if current term > request term, then tell the candidate that it is obsolete
	response.Term = gs.NodeRef.NodeContextInstance.CurrentTerm
	gs.NodeRef.NodeLogger.Infof("The Candidate Votes response is %+v", response)
	return response, nil
}

func (gs *GrpcServerImpl) RecoverEntriesService(ctx context.Context,
	request *pb.RecoverEntriesRequest) (*pb.RecoverEntriesResponse, error) {

	return &pb.RecoverEntriesResponse{
		Term:           0,
		NodeId:         0,
		PrevEntryIndex: 0,
		PrevEntryTerm:  0,
		Recovered:      false,
		EntryList:      nil,
	}, nil
}

// should input the self
func (gs *GrpcServerImpl) StartService() error {

	// get the listing address
	address := gs.NodeRef.NodeConfigInstance.Network.SelfAddress
	splittedAddress := strings.Split(address, ":")
	if len(splittedAddress) != 2 {
		return GRPCServerAddressError
	}

	// listening at a specific port
	listener, err := net.Listen("tcp", ":" + splittedAddress[1])
	if err != nil {
		return err
	}
	// start a new grpc server
	gs.grpcServer = grpc.NewServer()
	pb.RegisterRaftRPCServerServer(gs.grpcServer, gs)

	// start service
	err = gs.grpcServer.Serve(listener)
	if err != nil {
		return err
	}
	return nil
}

