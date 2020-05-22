package node

import (
	"errors"
	"fmt"
	"github.com/dlock_raft/protobuf"
	"github.com/dlock_raft/storage"
	"github.com/dlock_raft/utils"
	"math/rand"
	"os"
	"sync"
	"time"
)

const (
	// the fixed path for yaml format config file
	ConfigYamlFilePath = "../config/config.yaml"
)

var ReadConfigYamlError = errors.New("dlock_raft.init_node: Read yaml config error")
var ConstructLoggerError = errors.New("dlock_raft.init_node: Construct logger error")

type Node struct{

	// the instance for node config, read from config.yaml
	NodeConfigInstance *NodeConfig

	// the node log handler
	NodeLogger *utils.Logger

	// the node context
	NodeContextInstance *NodeContext

	// the in-memory state maps
	StateMapKVStore *storage.StateMapMemoryKVStore
	StateMapDLock *storage.StateMapMemoryDLock

	// the in-memory logEntry
	LogEntryInMemory *storage.LogMemory

	// all peers
	PeerList []*PeerNode

	// mutex for node object
	mutex *sync.RWMutex
}

type NodeOperators interface {

	BecomeFollower(term uint64)
	BecomeLeader()
	RandomElectionTimeout() time.Duration
	RunElectionDetectorModule()
	StartCandidateVoteModule()
	SendAppendEntriesToAllPeers(peerList []uint32)

}

func NewNode() (*Node, error){

	// read node config from yaml file
	nodeConfigInstance, err := NewNodeConfigFromYaml(ConfigYamlFilePath)
	if err != nil{
		return nil, ReadConfigYamlError
	}

	// new log object, with log path read in node config
	currentTimeString := time.Now().Format("20060102-150405")
	logFileName := nodeConfigInstance.Storage.LogPath + "Dlock-" + currentTimeString + ".log"
	logFileHandler, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return nil, ConstructLoggerError
	}
	nodeLoggerInstance, err := utils.New("DLock-Raft-Node", 1, logFileHandler)
	if err != nil {
		fmt.Println(err)
		return nil, ConstructLoggerError
	}

	// construct a new node object
	node := new(Node)
	node.NodeConfigInstance = nodeConfigInstance
	node.NodeLogger = nodeLoggerInstance

	return node, nil
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
	n.mutex.Unlock()

	// the ticker, every 10 ms ticks once
	ticker := time.NewTicker(15 * time.Millisecond)
	defer ticker.Stop()
	for {
		// trigger the following functionality every 10ms
		<-ticker.C

		n.mutex.Lock()
		// state has changed to leader or dead, jump out of election module, as it won't become candidate
		if n.NodeContextInstance.NodeState != Candidate && n.NodeContextInstance.NodeState != Follower {
			n.NodeLogger.Infof("Election timer has found a state change: %d", n.NodeContextInstance.NodeState)
			n.mutex.Unlock()
			return
		}

		// if term changes, also jump out of election module
		if n.NodeContextInstance.CurrentTerm > startElectionTerm {
			n.NodeLogger.Infof("Election timer has found a term change: %d", n.NodeContextInstance.CurrentTerm)
			n.mutex.Unlock()
			return
		}

		// if the time experienced exceeds election timeout, begin an election module
		if timeExperienced := time.Since(n.NodeContextInstance.electionRestartTime); timeExperienced >= electionTimeout{
			n.StartCandidateVoteModule()
			n.mutex.Unlock()
			return
		}
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
	n.NodeContextInstance.electionRestartTime = time.Now()
	n.NodeLogger.Infof("Begin the election module with term %d", n.NodeContextInstance.CurrentTerm)

	// the collected vote number and map
	collectedVote := 1
	voteMap := make(map[uint32]bool)
	voteMap[n.NodeConfigInstance.Id.SelfId] = true

	// create goroutine for every single peer
	for _, peer := range n.PeerList {
		voteMap[peer.PeerId] = false
		go func(peerObj *PeerNode) {
			n.mutex.Lock()
			// get the local maximum index and corresponding term
			maximumEntry, err := n.LogEntryInMemory.FetchLogEntry(n.LogEntryInMemory.MaximumIndex())
			if err != nil {
				n.NodeLogger.Errorf("Get entry with maximum index fails, maximum index: %d",
					n.LogEntryInMemory.MaximumIndex())
				n.mutex.Unlock()
				return
			}
			request := &protobuf.CandidateVotesRequest{
				Term:           n.NodeContextInstance.CurrentTerm,
				NodeId:         n.NodeConfigInstance.Id.SelfId,
				PrevEntryIndex: maximumEntry.Entry.Index,
				PrevEntryTerm:  maximumEntry.Entry.Term,
			}
			n.NodeLogger.Infof("The Candidate Vote request: %+v", request)
			// release mutex before sending GRPC request
			n.mutex.Unlock()

			// GPRC for Candidate Votes
			response, err := peerObj.GrpcClient.SendGrpcCandidateVotes(request)
			if err != nil {
				n.NodeLogger.Errorf("Send GRPC Candidate Vote fails, error: %s", err)
				return
			}
			n.NodeLogger.Infof("Get response from Candidate Vote, %+v", response)

			n.mutex.Lock()
			defer n.mutex.Unlock()
			// state has already changed
			if n.NodeContextInstance.NodeState != Candidate {
				n.NodeLogger.Infof("During waiting Candidate Vote response, state changes to %d",
					n.NodeContextInstance.NodeState)
				return
			}
			// term has increased
			// note that the term to compared is the term when StartCandidateVoteModule starts
			if response.Term > savedCurrentTerm {
				n.NodeLogger.Infof("During waiting Candidate Vote response, term changes to %d",
					response.Term)
				n.BecomeFollower(response.Term)
				return
			} else if response.Term == savedCurrentTerm && n.NodeContextInstance.CurrentTerm == savedCurrentTerm {
				// if the candidate vote is still in time
				if response.Accepted == true && voteMap[peerObj.PeerId] == false {
					collectedVote += 1
				}
				// if collected vote exceeds n/2 + 1, then become a leader
				if 2 * collectedVote > len(n.NodeConfigInstance.Id.PeerId) {
					n.NodeLogger.Infof("Candidate Vote collects %d votes in term %d, become leader",
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
	n.NodeContextInstance.electionRestartTime = time.Now()
	go n.RunElectionDetectorModule()
}

// become leader
func (n *Node) BecomeLeader() {
	// note that this function can only be entered in RunElectionDetectorModule
	// thus, n.mutex has already been locked

	n.NodeLogger.Infof("Node become leader in term %d.", n.NodeContextInstance.CurrentTerm)
	n.NodeContextInstance.NodeState = Leader
	for _, peer := range n.PeerList {
		// start to search for last common entry index with every peer, beginning from the current maximum index + 1
		peer.NextIndex = n.LogEntryInMemory.MaximumIndex() + 1
	}

	// the timer, tick interval is determined in config.yaml
	// for every interval, if no AppendEntries is sent, then send heartbeat (empty AppendEntries)
	// every time AppendEntries is sent, reset the timer
	ticker := time.NewTimer(time.Duration(n.NodeConfigInstance.Parameters.HeartBeatInterval) * time.Millisecond)
	defer ticker.Stop()

	// start a new go routine for hear beat
	go func() {
		for {
			// the sendTag indicates whether the leader should trigger AE to every follower
			sendTag := false

			select {
			// if tick time exceeds heartbeat interval
			case <-ticker.C:
				sendTag = true
				// Reset timer
				ticker.Stop()
				ticker.Reset(time.Duration(n.NodeConfigInstance.Parameters.HeartBeatInterval) * time.Millisecond)
			// if semaphore for send AppendEntries is triggered
			case _, ok := <-n.NodeContextInstance.AppendEntryChan:
				if ok == true {
					sendTag = true
				} else {
					return
				}
				// Reset timer
				ticker.Stop()
				ticker.Reset(time.Duration(n.NodeConfigInstance.Parameters.HeartBeatInterval) * time.Millisecond)
			}

			// if get the sendTag, then send AppendEntries
			if sendTag {
				n.mutex.Lock()
				if n.NodeContextInstance.CurrentTerm == Leader {
					n.mutex.Unlock()
					n.SendAppendEntriesToPeers(nil)
				} else {
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
	// local term when begin to sen AppendEntries
	startCurrentTerm:= n.NodeContextInstance.CurrentTerm

	for i, peer := range n.PeerList {
		// peerList == nil means send to all peers
		// if peerList != nil, only send to peers in peerList
		if peerList != nil && !utils.NumberInUint32List(peerList, peer.PeerId) {
			continue
		}
		indexIntermediate := i
		go func() {
			n.mutex.Lock()
			// get out of the module when finding that the node is not leader
			if n.NodeContextInstance.NodeState != Leader {
				n.NodeLogger.Infof("Before sending AppendEntries to %d, state has changed to %d.",
					n.PeerList[indexIntermediate].PeerId, n.NodeContextInstance.NodeState)
				n.mutex.Unlock()
				return
			}
			// for this time, begin from index of nextIndex - 1
			prevIndex := n.PeerList[indexIntermediate].NextIndex - 1
			maximumIndex := n.LogEntryInMemory.MaximumIndex()
			// note that entry list begins from nextIndex, and length new never exceeds maximumEntryListLength
			entryLength := utils.Uint64Min(maximumEntryListLength, maximumIndex - prevIndex)

			// fill in the entry list for append
			entryList := make([]*protobuf.Entry, entryLength)
			for j := prevIndex + 1; j <= prevIndex + entryLength; j ++ {
				logEntry, err := n.LogEntryInMemory.FetchLogEntry(j)
				if err != nil {
					n.NodeLogger.Errorf("Error happens when fetching LogEntry %d, error: %s", j, err)
					n.mutex.Unlock()
					return
				}
				entryList[j - prevIndex - 1] = logEntry.Entry
			}

			// get prevTerm
			logEntryPrev, err := n.LogEntryInMemory.FetchLogEntry(prevIndex)
			if err != nil {
				n.NodeLogger.Errorf("Error happens when fetching LogEntry %d, error: %s", prevIndex, err)
				n.mutex.Unlock()
				return
			}

			// construct the request
			request := &protobuf.AppendEntriesRequest{
				Term:             n.NodeContextInstance.CurrentTerm,
				NodeId:           n.NodeConfigInstance.Id.SelfId,
				PrevEntryIndex:   prevIndex,
				PrevEntryTerm:    logEntryPrev.Entry.Term,
				CommitEntryIndex: n.NodeContextInstance.CommitIndex,
				EntryList:        entryList,
			}
			// unlock before send the request
			n.mutex.Unlock()

			response, err := n.PeerList[indexIntermediate].GrpcClient.SendGrpcAppendEntries(request)
			if err != nil {
				n.NodeLogger.Errorf("Send GRPC AppendEntries fails, error: %s.", err)
				return
			}
			n.NodeLogger.Infof("Get response from AppendEntries, %+v.", response)

			// now begin to process the response
			n.mutex.Lock()
			// get out of the leader module when finding that the node is not leader
			if n.NodeContextInstance.NodeState != Leader {
				n.NodeLogger.Infof("Before getting AppendEntries response from %d, state has changed to %d.",
					n.PeerList[indexIntermediate].PeerId, n.NodeContextInstance.NodeState)
				n.mutex.Unlock()
				return
			}
			// become follower if the remote has higher term number
			if response.Term > startCurrentTerm {
				n.NodeLogger.Infof("Receiving AppendEntries response, but remote term of %d has changed to %d.",
					response.NodeId, response.Term)
				n.BecomeFollower(response.Term)
				n.mutex.Unlock()
				return
			}
			// still a valid leader, then process the response
			if response.Term == startCurrentTerm{
				if response.Success {

				}
			}


		}()
	}

}

