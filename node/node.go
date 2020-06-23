package node

import (
	"errors"
	"fmt"
	"github.com/dlock_raft/protobuf"
	"github.com/dlock_raft/storage"
	"github.com/dlock_raft/utils"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"time"
)

const (
	// the default path for yaml format config file
	// often used for docker deployment
	DefaultConfigYamlFilePath = "config/config.yaml"
)

var ReadConfigYamlError = errors.New("dlock_raft.init_node: Read yaml config error")
var ConstructLoggerError = errors.New("dlock_raft.init_node: Construct logger error")
var InitContextError = errors.New("dlock_raft.init_node: Init node context error")
var RecoverLogMemoryError = errors.New("dlock_raft.init_node: Recover log memory error")
var InitStateMapError = errors.New("dlock_raft.init_node: Init state map error")
var InitPeerListError = errors.New("dlock_raft.init_node: Init peer list error")
var InitP2PServerError = errors.New("dlock_raft.init_node: Init Peer2Peer GRPC server error")
var InitCliSrvServerError = errors.New("dlock_raft.init_node: Init Cli-Srv GRPC server error")

type Node struct{

	// the instance for node config, read from config.yaml
	NodeConfigInstance *NodeConfig

	// the node log handler
	NodeLogger *utils.Logger

	// the node context
	NodeContextInstance *NodeContext

	// the node P2P server
	NodeServer *GrpcP2PServerImpl
	// the node cli-svr server
	CliServer *GRPCCliSrvServerImpl

	// the in-memory state maps
	StateMapKVStore *storage.StateMapMemoryKVStore
	StateMapDLock *storage.StateMapMemoryDLock

	// the in-memory logEntry
	LogEntryInMemory *storage.LogMemory

	// all peers
	PeerList []*PeerNode

	// dlock interchange, available only if node state is Leader
	DlockInterchangeInstance *DlockInterchange

	// mutex for node object
	mutex sync.Mutex
}

type NodeOperators interface {

	BecomeFollower(term uint64)
	BecomeLeader()
	RandomElectionTimeout() time.Duration
	RunElectionDetectorModule()
	StartCandidateVoteModule()
	SendAppendEntriesToPeers(peerIdList []uint32)
	CommitToStateMap()
	BackUpLogMemoryToDisk()
	InitRaftConsensusModule()

}

// the dlock peer node initialization function
// your can revise config file path with an optional parameter
func NewNode(configFile ...string) (*Node, error){

	// read node config from yaml file
	var actualConfigFilePath string
	if len(configFile) >= 1 && len(configFile[0]) != 0 {
		actualConfigFilePath = configFile[0]
	} else {
		actualConfigFilePath = DefaultConfigYamlFilePath
	}
	nodeConfigInstance, err := NewNodeConfigFromYaml(actualConfigFilePath)
	if err != nil {
		return nil, ReadConfigYamlError
	}
	// new log object, with log path read in node config
	currentTimeString := time.Now().Format("20060102-150405")
	logFileName := nodeConfigInstance.Storage.LogPath + "Dlock-" +
		currentTimeString + "-" + strconv.Itoa(int(nodeConfigInstance.Id.SelfId)) + ".log"
	logFileHandler, err2 := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err2 != nil {
		return nil, err2
	}
	// the logger for node
	nodeLoggerInstance, err3 := utils.New("DLock-Raft-Node", 1, logFileHandler)
	if err3 != nil {
		return nil, ConstructLoggerError
	}
	// set log level as DEBUG
	nodeLoggerInstance.SetLogLevel(utils.DebugLevel)
	nodeLoggerInstance.Info("Reconstruct Node Logger succeeded.")

	// the initial context for node
	nodeContextInstance, err4 := NewStartNodeContext(nodeConfigInstance)
	if err4 != nil {
		nodeLoggerInstance.Errorf("Init node context fails, error: %s.", err4)
		return nil, InitContextError
	}
	nodeLoggerInstance.Info("Init node context succeeded.")

	// init or construct an in-memory LogEntry List
	logMemory := storage.NewLogMemory()
	readBytes, err5 := logMemory.LogReload(nodeContextInstance.DiskLogEntry)
	if err5 != nil {
		nodeLoggerInstance.Errorf("Reconstruct in-memory LogMemory fails, error: %s.", err5)
		return nil, RecoverLogMemoryError
	}
	// init some context parameters according to the backup logEntry
	nodeContextInstance.LastBackupIndex = logMemory.MaximumIndex()
	nodeContextInstance.CommitIndex = logMemory.MaximumIndex()
	nodeContextInstance.LastAppliedIndex = logMemory.MaximumIndex()
	if logMemory.MaximumIndex() != 0 {
		lastEntry, err := logMemory.FetchLogEntry(logMemory.MaximumIndex())
		if err != nil {
			nodeLoggerInstance.Errorf("Read the Last LogEntry fails, entry index %d", logMemory.MaximumIndex())
			return nil, RecoverLogMemoryError
		}
		nodeContextInstance.CurrentTerm = lastEntry.Entry.Term
	}
	nodeLoggerInstance.Infof("Reconstruct in-memory LogMemory succeeded, read %d bytes.", readBytes)

	// init state map in memory from LogMemory
	stateMapDLock, err6 := storage.NewStateMapMemoryDLock("DLock")
	stateMapKVStore, err7 := storage.NewStateMapMemoryKVStore("KVStore")
	if err6 != nil || err7 != nil {
		nodeLoggerInstance.Errorf("Init state map fails, error: %s, %s.", err6, err7)
		return nil, InitStateMapError
	}
	// update state map by LogMemory
	if logMemory.MaximumIndex() >= 1 {
		err8 := stateMapDLock.UpdateStateFromLogMemory(logMemory, 1, logMemory.MaximumIndex())
		err9 := stateMapKVStore.UpdateStateFromLogMemory(logMemory, 1, logMemory.MaximumIndex())
		if err8 != nil || err9 != nil {
			nodeLoggerInstance.Errorf("Update state map from db fails, error: %s, %s.", err8, err9)
			return nil, InitStateMapError
		}
	} else {
		nodeLoggerInstance.Infof("Creating StateMaps from scratch, no available entries.")
	}

	// construct a new node object
	node := &Node{
		NodeConfigInstance:  nodeConfigInstance,
		NodeLogger:          nodeLoggerInstance,
		NodeContextInstance: nodeContextInstance,
		StateMapKVStore:     stateMapKVStore,
		StateMapDLock:       stateMapDLock,
		LogEntryInMemory:    logMemory,
	}

	// init peer node objects, with node config existing
	peerList, err10 := NewPeerNodeListFromConfig(node)
	if err10 != nil {
		nodeLoggerInstance.Errorf("New Peer list init fails, error: %s.", err10)
		return nil, InitPeerListError
	}
	node.PeerList = peerList

	// start the raft server for P2P, with node config existing
	node.NodeServer, err = NewGrpcP2PServerImpl(node)
	if err != nil {
		nodeLoggerInstance.Errorf("New P2P transportation server init fails, error: %s.", err10)
		return nil, InitP2PServerError
	}

	// start the raft server for cli-srv, with node config existing
	node.CliServer, err = NewGRPCCliSrvServerImpl(node)
	if err != nil {
		nodeLoggerInstance.Errorf("New cli-srv transportation server init fails, error: %s.", err10)
		return nil, InitCliSrvServerError
	}
	return node, nil

}

// the entry point of raft-based dlock
func (n* Node) InitRaftConsensusModule() {
	// begin as a Follower, waiting to enter election
	if n.NodeContextInstance.NodeState == Dead {
		n.NodeContextInstance.NodeState = Follower
	}
	go n.NodeServer.StartService()
	go n.RunElectionDetectorModule()
	go n.BackUpLogMemoryToDisk()
	go n.CommitToStateMap()
	go n.CliServer.StartService()
}

// get a random election timeout
// ranging from MinWaitTimeCandidate to MaxWaitTimeCandidate
func (n *Node) RandomElectionTimeout() time.Duration {

	minWaitTimeCandidate := n.NodeConfigInstance.Parameters.MinWaitTimeCandidate
	maxWaitTimeCandidate := n.NodeConfigInstance.Parameters.MaxWaitTimeCandidate
	return time.Duration(int(minWaitTimeCandidate) +
		rand.Intn(int(maxWaitTimeCandidate-minWaitTimeCandidate))) * time.Millisecond
}

// the election module
func (n *Node) RunElectionDetectorModule() {
	// the random election timeout
	electionTimeout := n.RandomElectionTimeout()
	n.mutex.Lock()
	startElectionTerm := n.NodeContextInstance.CurrentTerm
	// refresh the election restart time, very important !
	n.NodeContextInstance.ElectionRestartTime = time.Now()
	n.mutex.Unlock()
	n.NodeLogger.Infof("Start Election Module for Term %d, election timeout %s.\n",
		startElectionTerm, electionTimeout)

	// the ticker, every 10 ms ticks once
	ticker := time.NewTicker(15 * time.Millisecond)
	defer ticker.Stop()
	for {
		// trigger the following functionality every 10ms
		<-ticker.C

		n.mutex.Lock()
		// state has changed to leader or dead, jump out of election module, as it won't become candidate
		if n.NodeContextInstance.NodeState != Candidate && n.NodeContextInstance.NodeState != Follower {
			n.NodeLogger.Debugf("Election timer has found a state change: %d.", n.NodeContextInstance.NodeState)
			n.mutex.Unlock()
			return
		}

		// if term changes, also jump out of election module
		if n.NodeContextInstance.CurrentTerm > startElectionTerm {
			n.NodeLogger.Debugf("Election timer has found a term change: %d.", n.NodeContextInstance.CurrentTerm)
			n.mutex.Unlock()
			return
		}

		// if the time experienced exceeds election timeout, begin an election module
		timeExperienced := time.Since(n.NodeContextInstance.ElectionRestartTime)
		if timeExperienced >= electionTimeout{
			n.NodeLogger.Debugf("Entering the candidate vote module, Current Term %d," +
				" time experienced %s.", startElectionTerm, timeExperienced)
			n.StartCandidateVoteModule()
			n.mutex.Unlock()
			return
		}
		n.mutex.Unlock()
	}
}

// entering this module, then the node enters the Candidate State
// the node will send CandidateVotes to every other peer, in order to become a leader
// every Candidate vote request will be processed by a single goroutine
func (n *Node) StartCandidateVoteModule() {
	// note that this function can only be entered in RunElectionDetectorModule
	// thus, n.mutex has already been locked

	// change the state to Candidate
	n.NodeContextInstance.NodeState = Candidate
	// increase current term
	n.NodeContextInstance.CurrentTerm += 1
	savedCurrentTerm := n.NodeContextInstance.CurrentTerm
	// vote for itself
	n.NodeContextInstance.VotedPeer = n.NodeConfigInstance.Id.SelfId
	// reset the election time ticker
	n.NodeContextInstance.ElectionRestartTime = time.Now()
	// reset hop
	n.NodeContextInstance.HopToCurrentLeaderId = 0

	n.NodeLogger.Debugf("Begin the Candidate Vote module with term %d", n.NodeContextInstance.CurrentTerm)

	// the collected vote number and map
	collectedVote := 1
	voteMap := make(map[uint32]bool)
	voteMap[n.NodeConfigInstance.Id.SelfId] = true

	// create goroutine for every single peer for this term
	for _, peer := range n.PeerList {
		voteMap[peer.PeerId] = false
		go func(peerObj *PeerNode) {
			n.mutex.Lock()

			// state has changed to leader or dead, jump out of election module, as it won't become candidate
			if n.NodeContextInstance.NodeState != Candidate && n.NodeContextInstance.NodeState != Follower {
				n.NodeLogger.Debugf("Candidate Vote module has found a state change: %d",
					n.NodeContextInstance.NodeState)
				n.mutex.Unlock()
				return
			}
			// if term changes, also jump out of election module
			if n.NodeContextInstance.CurrentTerm > savedCurrentTerm {
				n.NodeLogger.Debugf("Candidate Vote module has found a term change: from %d to %d",
					savedCurrentTerm, n.NodeContextInstance.CurrentTerm)
				n.mutex.Unlock()
				return
			}

			// get the local maximum index and corresponding term
			maximumEntryTerm := uint64(0)
			if n.LogEntryInMemory.MaximumIndex() != 0 {
				maximumEntry, err := n.LogEntryInMemory.FetchLogEntry(n.LogEntryInMemory.MaximumIndex())
				if err != nil {
					n.NodeLogger.Errorf("Get entry with maximum index fails, maximum index: %d",
						n.LogEntryInMemory.MaximumIndex())
					n.mutex.Unlock()
					return
				}
				maximumEntryTerm = maximumEntry.Entry.Term
			}
			// construct request for Candidate Vote
			request := &protobuf.CandidateVotesRequest{
				Term:           savedCurrentTerm,
				NodeId:         n.NodeConfigInstance.Id.SelfId,
				PrevEntryIndex: n.LogEntryInMemory.MaximumIndex(),
				PrevEntryTerm:  maximumEntryTerm,
			}
			n.NodeLogger.Debugf("The Candidate Vote request: %+v", request)
			// release mutex before sending GRPC request
			n.mutex.Unlock()

			// GPRC for Candidate Votes
			response, err := peerObj.GrpcClient.SendGrpcCandidateVotes(request)
			if err != nil {
				n.NodeLogger.Errorf("Send GRPC Candidate Vote fails, error: %s", err)
				return
			}
			n.NodeLogger.Debugf("Get response from Candidate Vote, %+v", response)

			n.mutex.Lock()
			defer n.mutex.Unlock()
			// state has already changed
			if n.NodeContextInstance.NodeState != Candidate {
				n.NodeLogger.Debugf("During waiting Candidate Vote response, state changes to %d",
					n.NodeContextInstance.NodeState)
				return
			}
			// term has increased
			// note that the term to compared is the term when StartCandidateVoteModule starts
			if response.Term > savedCurrentTerm {
				n.NodeLogger.Debugf("During waiting CandidateVote response, term changes to %d",
					response.Term)
				n.NodeContextInstance.HopToCurrentLeaderId = 0
				n.BecomeFollower(response.Term)
				return
			} else if response.Term == savedCurrentTerm && n.NodeContextInstance.CurrentTerm == savedCurrentTerm {
				// if the candidate vote is still in time
				if response.Accepted == true && voteMap[peerObj.PeerId] == false {
					collectedVote += 1
					voteMap[peerObj.PeerId] = true
					n.NodeLogger.Debugf("Receive successful CandidateVote response from node %d, now " +
						"have %d votes in term %d", response.NodeId, collectedVote, response.Term)
				}
				// if collected vote exceeds n/2 + 1, then become a leader
				if 2 * collectedVote > len(n.NodeConfigInstance.Id.PeerId) + 1 {
					n.NodeLogger.Debugf("Node collects %d CandidateVote response in term %d, now become leader.",
						collectedVote, response.Term)
					n.BecomeLeader()
					return
				}

			}

		}(peer)
	}
	// in case the above module fails
	go n.RunElectionDetectorModule()
	// note that mutex will be unlocked in RunElectionDetectorModule
}

// become follower
func (n *Node) BecomeFollower(term uint64) {

	n.NodeLogger.Infof("Node become follower in term %d.", n.NodeContextInstance.CurrentTerm)
	// 0 means vote for nobody
	n.NodeContextInstance.VotedPeer = 0
	n.NodeContextInstance.CurrentTerm = term
	n.NodeContextInstance.NodeState = Follower

	// start a new election timeout goroutine
	n.NodeContextInstance.ElectionRestartTime = time.Now()
	go n.RunElectionDetectorModule()
}

// become leader
func (n *Node) BecomeLeader() {
	// note that this function can only be entered in RunElectionDetectorModule
	// thus, n.mutex has already been locked

	n.NodeLogger.Infof("Node become leader in term %d.", n.NodeContextInstance.CurrentTerm)
	n.NodeContextInstance.NodeState = Leader
	// when first become a leader, create a new dlockInterchange
	n.DlockInterchangeInstance = NewDlockInterchange(n)
	err := n.DlockInterchangeInstance.InitFromDLockStateMap(time.Now().UnixNano())
	if err != nil {
		n.NodeLogger.Errorf("Init DLock Interchange fails, error %s.", err)
	}
	// start a new goroutine to monitor the DLocks
	go n.DlockInterchangeInstance.ReleaseExpiredDLockPeriodically()

	for _, peer := range n.PeerList {
		// start to search for last common entry index with every peer, beginning from the current maximum index + 1
		peer.NextIndex = n.LogEntryInMemory.MaximumIndex() + 1
	}

	// the timer, tick interval is determined in config.yaml
	// for every interval, if no AppendEntries is sent, then send heartbeat (empty AppendEntries)
	// every time AppendEntries is sent, reset the timer
	timer := time.NewTimer(time.Duration(n.NodeConfigInstance.Parameters.HeartBeatInterval) * time.Millisecond)

	// start a new go routine for hear beat
	go func() {
		for {
			// the sendTag indicates whether the leader should trigger AE to every follower
			sendTag := false

			select {
			// if tick time exceeds heartbeat interval
			case <-timer.C:
				n.NodeLogger.Debugf("Sending AppendEntries to all peers is triggered by heartbeat, term %d",
					n.NodeContextInstance.CurrentTerm)
				sendTag = true
				// Reset timer
				timer.Stop()
				timer.Reset(time.Duration(n.NodeConfigInstance.Parameters.HeartBeatInterval) * time.Millisecond)

			// or if semaphore for sending AppendEntries is triggered deliberately
			case <-n.NodeContextInstance.AppendEntryChan:
				n.NodeLogger.Debugf("Sending AppendEntries to all peers is triggered by new Entries, term %d",
					n.NodeContextInstance.CurrentTerm)
				sendTag = true
				// Reset timer
				timer.Stop()
				timer.Reset(time.Duration(n.NodeConfigInstance.Parameters.HeartBeatInterval) * time.Millisecond)
			}

			fmt.Println("Leader Heat Beat.")

			// if get the sendTag, then send AppendEntries
			if sendTag {
				n.mutex.Lock()
				if n.NodeContextInstance.NodeState == Leader {
					n.NodeLogger.Debugf("Now begin to send AppendEntries to all peers, term %d",
						n.NodeContextInstance.CurrentTerm)
					n.mutex.Unlock()
					n.SendAppendEntriesToPeers(nil)
				} else {
					timer.Stop()
					n.mutex.Unlock()
					return
				}
			}
		}
	}()
	// note that mutex will be unlocked in RunElectionDetectorModule
}

// send AppendEntries to all peers
// peerList is used to indicate which peers should be sent
// nil used to indicate all peers
func (n *Node) SendAppendEntriesToPeers(peerList []uint32) {
	n.mutex.Lock()
	defer n.mutex.Unlock()
	// the maximum number of LogEntries appended each time
	maximumEntryListLength := uint64(n.NodeConfigInstance.Parameters.MaxLogUnitsRecover)
	// local term when begin to send AppendEntries
	startCurrentTerm:= n.NodeContextInstance.CurrentTerm

	for i, peer := range n.PeerList {
		// peerList == nil means send to all peers
		// if peerList != nil, only send to peers in peerList
		if peerList != nil && !utils.NumberInUint32List(peerList, peer.PeerId) {
			n.NodeLogger.Debugf("When sending AppendEntries, Node %d is skipped.", peer.PeerId)
			continue
		}
		indexIntermediate := i
		n.NodeLogger.Debugf("Trigger a goroutine for AppendEntries to node %d at Term %d",
			peer.PeerId, startCurrentTerm)
		go func() {
			n.mutex.Lock()
			// get out of the module when finding that the node is not leader
			if n.NodeContextInstance.NodeState != Leader {
				n.NodeLogger.Debugf("Before sending AppendEntries to %d, state has changed to %d.",
					n.PeerList[indexIntermediate].PeerId, n.NodeContextInstance.NodeState)
				n.mutex.Unlock()
				return
			}
			// for this time, begin from index of nextIndex - 1
			nextIndexRecord := n.PeerList[indexIntermediate].NextIndex
			prevIndexRecord := nextIndexRecord - 1
			maximumIndex := n.LogEntryInMemory.MaximumIndex()
			// note that entry list begins from nextIndex, and length new never exceeds maximumEntryListLength
			entryLength := utils.Uint64Min(maximumEntryListLength, maximumIndex - prevIndexRecord)
			n.NodeLogger.Debugf("Before sending AppendEntries to peer %d, nextIndex: %d," +
				" prevIndex: %d, maximumIndex: %d, length of entry to be attached: %d",
				n.PeerList[indexIntermediate].PeerId, nextIndexRecord, prevIndexRecord, maximumIndex, entryLength)

			// fill in the entry list for append
			entryList := make([]*protobuf.Entry, entryLength)
			for j := prevIndexRecord + 1; j <= prevIndexRecord + entryLength; j ++ {
				logEntry, err := n.LogEntryInMemory.FetchLogEntry(j)
				if err != nil {
					n.NodeLogger.Errorf("Error happens when fetching LogEntry %d, error: %s", j, err)
					n.mutex.Unlock()
					return
				}
				entryList[j - prevIndexRecord - 1] = logEntry.Entry
			}

			// get prevTerm
			prevEntryTerm := uint64(0)
			if prevIndexRecord != 0{
				logEntryPrev, err := n.LogEntryInMemory.FetchLogEntry(prevIndexRecord)
				if err != nil {
					n.NodeLogger.Errorf("Error happens when fetching LogEntry %d, error: %s", prevIndexRecord, err)
					n.mutex.Unlock()
					return
				}
				prevEntryTerm = logEntryPrev.Entry.Term
			}

			// construct the request
			request := &protobuf.AppendEntriesRequest{
				Term:             n.NodeContextInstance.CurrentTerm,
				NodeId:           n.NodeConfigInstance.Id.SelfId,
				PrevEntryIndex:   prevIndexRecord,
				PrevEntryTerm:    prevEntryTerm,
				CommitEntryIndex: n.NodeContextInstance.CommitIndex,
				EntryList:        entryList,
			}
			// unlock before send the request by GRPC
			n.mutex.Unlock()

			n.NodeLogger.Debugf("Sending AppendEntries to peer %d, %+v.",
				n.PeerList[indexIntermediate].PeerId, request)
			response, err := n.PeerList[indexIntermediate].GrpcClient.SendGrpcAppendEntries(request)
			if err != nil {
				n.NodeLogger.Errorf("Send GRPC AppendEntries fails, error: %s.", err)
				return
			}
			n.NodeLogger.Debugf("Get response from AppendEntries, %+v.", response)

			// now begin to process the response
			n.mutex.Lock()
			// get out of the leader module when finding that the node is not leader
			if n.NodeContextInstance.NodeState != Leader {
				n.NodeLogger.Debugf("Before getting AppendEntries response from %d, state has changed to %d.",
					n.PeerList[indexIntermediate].PeerId, n.NodeContextInstance.NodeState)
				n.mutex.Unlock()
				return
			}
			// become follower if the remote has higher term number
			if response.Term > startCurrentTerm {
				n.NodeLogger.Debugf("Receiving AppendEntries response, but remote term of %d has changed to %d.",
					response.NodeId, response.Term)
				n.NodeContextInstance.HopToCurrentLeaderId = 0
				n.BecomeFollower(response.Term)
				n.mutex.Unlock()
				return
			}
			// if still a valid leader, then process the response
			if response.Term == startCurrentTerm {

				if response.Success {
					if prevIndexRecord+1 <= prevIndexRecord+entryLength {
						n.NodeLogger.Debugf("Peer %d append entry from %d to %d succeeded.", response.NodeId,
							prevIndexRecord+1, prevIndexRecord+entryLength)
					} else {
						n.NodeLogger.Debugf("Peer %d append no entries (for heartbeat).", response.NodeId)
					}

					// update nextIndex
					n.PeerList[indexIntermediate].NextIndex = nextIndexRecord + entryLength
					// if prevIndex matches in peers' local LogMemory (Success == true)
					// then update matchIndex equal to the index of the last appended LogEntry
					if n.PeerList[indexIntermediate].NextIndex-1 > n.PeerList[indexIntermediate].MatchIndex {
						n.PeerList[indexIntermediate].MatchIndex = n.PeerList[indexIntermediate].NextIndex - 1
					}

					origCommitIndex := n.NodeContextInstance.CommitIndex
					// find whether there is an LogEntry that has been logged in LogMemory by majority of peers
					for k := origCommitIndex + 1; k <= n.LogEntryInMemory.MaximumIndex(); k++ {
						// count itself first
						matchedPeers := 1
						// count the followers
						for _, peerInstance := range n.PeerList {
							if peerInstance.MatchIndex >= k {
								matchedPeers += 1
							}
						}
						// if majority (>n/2+1) of peers commit, then itself commit
						if 2*matchedPeers > len(n.NodeConfigInstance.Id.PeerId)+1 {
							n.NodeContextInstance.CommitIndex = k
							n.NodeLogger.Debugf("Majority of peers append LogEntry "+
								"with index %d, now can commit it.", k)
						}
					}

					// if some update to commitIndex happened, then start the commit goroutine
					if n.NodeContextInstance.CommitIndex > origCommitIndex {
						n.NodeLogger.Debugf("Commitment is to be triggered by AppendEntries success in term %d, "+
							"original commit index %d, to be committed index %d.",
							startCurrentTerm, origCommitIndex, n.NodeContextInstance.CommitIndex)
						// begin to update statemap
						n.NodeContextInstance.TriggerCommitChannel()
						// tell other followers for commitIndex update
						n.NodeContextInstance.TriggerAEChannel()
					}

				} else {
					// if appending LogEntry does not succeed, decrease nextIndex
					if response.ConflictEntryTerm > 0 {
						lastIndexOfTerm, err := n.LogEntryInMemory.FetchLastIndexOfTerm(response.ConflictEntryTerm)
						if err == storage.InMemoryNoSpecificTerm {
							n.PeerList[indexIntermediate].NextIndex = response.ConflictEntryIndex
						} else if err != nil {
							n.NodeLogger.Errorf("Error happens when searching certain term %d, error: %s",
								response.ConflictEntryTerm, err)
							n.mutex.Unlock()
							return
						} else {
							n.PeerList[indexIntermediate].NextIndex = lastIndexOfTerm + 1
							n.NodeLogger.Debugf("Set NextIndex of peer %d as lastIndexOfTerm + 1 (%d)",
								indexIntermediate, lastIndexOfTerm+1)
						}
					} else {
						// note that conflict entry index is often the first LogEntry leader sends
						// thus, setting it as nextIndex is a kind of decrement (NextIndex--)
						n.PeerList[indexIntermediate].NextIndex = response.ConflictEntryIndex
						n.NodeLogger.Debugf("Set NextIndex of peer %d as ConflictEntryIndex (%d)",
							indexIntermediate, response.ConflictEntryIndex)
					}
					n.NodeLogger.Debugf("The nextIndex of node %d has changed to %d",
						n.PeerList[indexIntermediate].PeerId, n.PeerList[indexIntermediate].NextIndex)

					// start a new goroutine to process remaining entry append work
					appendAgainList := []uint32{n.PeerList[indexIntermediate].PeerId}
					go n.SendAppendEntriesToPeers(appendAgainList)
				}
			}
			// don't forget to unlock mutex
			n.mutex.Unlock()
		}()
	}
}

// update stateMap when receiving a signal from CommitChan or NewLeaderCommitChan
// this function should be running throughout the lifecycle, in a goroutine
func (n *Node) CommitToStateMap() {

	for {
		select {
			// if semaphore for updating stateMap is triggered
			case _, ok := <-n.NodeContextInstance.CommitChan:
				if ok == true {
					n.mutex.Lock()
					n.commitProcedure()
					n.mutex.Unlock()
				}
		}
	}
}

// should enter with mutex
func (n *Node) commitProcedure() {
	// if new entries should be applied to stateMaps
	if n.NodeContextInstance.CommitIndex > n.NodeContextInstance.LastAppliedIndex{
		// update the two stateMaps in memory
		err := n.StateMapDLock.UpdateStateFromLogMemory(n.LogEntryInMemory,
			n.NodeContextInstance.LastAppliedIndex + 1, n.NodeContextInstance.CommitIndex)
		if err != nil {
			n.NodeLogger.Errorf("Update local DLock stateMap failed for entry from %d to %d, error: %s.",
				n.NodeContextInstance.LastAppliedIndex + 1, n.NodeContextInstance.CommitIndex, err)
		} else {
			n.NodeLogger.Debugf("Update local DLock stateMap succeeded for entry from %d to %d.",
				n.NodeContextInstance.LastAppliedIndex + 1, n.NodeContextInstance.CommitIndex)
		}
		err2 := n.StateMapKVStore.UpdateStateFromLogMemory(n.LogEntryInMemory,
			n.NodeContextInstance.LastAppliedIndex + 1, n.NodeContextInstance.CommitIndex)
		if err2 != nil {
			n.NodeLogger.Errorf("Update local KVStore stateMap fails for entry from %d to %d, error: %s.",
				n.NodeContextInstance.LastAppliedIndex + 1, n.NodeContextInstance.CommitIndex, err2)
		} else {
			n.NodeLogger.Debugf("Update local KVStore stateMap succeeded for entry from %d to %d.",
				n.NodeContextInstance.LastAppliedIndex + 1, n.NodeContextInstance.CommitIndex)
		}
		// don't forget to refresh LastAppliedIndex
		n.NodeContextInstance.LastAppliedIndex = n.NodeContextInstance.CommitIndex
	}
}


// backup the LogMemory in disk
// this function should be running throughout the lifecycle, in a goroutine
// the backup interval is set up in config.yaml
func (n *Node) BackUpLogMemoryToDisk() {

	// ticks every fixed interval
	ticker := time.NewTicker(time.Duration(n.NodeConfigInstance.Parameters.LogBackupInterval) * time.Millisecond)
	for {
		select {
		// if semaphore for updating stateMap is triggered
		case <- ticker.C:
			n.mutex.Lock()
			n.backupProcedure()
			n.mutex.Unlock()
		}
	}
}

// should enter with mutex
func (n *Node) backupProcedure() {
	// if new committed entries should be backup
	if n.NodeContextInstance.CommitIndex > n.NodeContextInstance.LastBackupIndex &&
		n.NodeContextInstance.DiskLogEntry != nil {
		// back up the LogEntries
		writeBytes, err := n.LogEntryInMemory.StoreLogMemory(n.NodeContextInstance.LastBackupIndex + 1,
			n.NodeContextInstance.CommitIndex, n.NodeContextInstance.DiskLogEntry)
		if err != nil {
			n.NodeLogger.Errorf("Backup LogMemory failed for entry from %d to %d, error: %s.",
				n.NodeContextInstance.LastBackupIndex + 1, n.NodeContextInstance.CommitIndex, err)
		} else {
			n.NodeLogger.Debugf("Backup LogMemory succeeded for entry from %d to %d, written %d bytes.",
				n.NodeContextInstance.LastBackupIndex + 1, n.NodeContextInstance.CommitIndex, writeBytes)
			n.NodeContextInstance.LastBackupIndex = n.NodeContextInstance.CommitIndex
		}
	}
}