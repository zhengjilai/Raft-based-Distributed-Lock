// an implementation for NodeContext
// all information in NodeContext are dynamic
// often related to the current node state, and is changing rapidly
package node

import (
	"os"
	"time"
)

const (
	Dead = iota
	Leader
	Candidate
	Follower
)

// NodeContext is the concrete implementation of NodeContext.
type NodeContext struct {

	// the current term
	CurrentTerm  uint64
	// the last committed index in local LogMemory
	CommitIndex  uint64
	
	// last applied entry index (to StateMap)
	LastAppliedIndex uint64
	// last backup entry index (to local file in disk)
	LastBackupIndex uint64
	
	// the node state
	NodeState int
	
	// the hop id for current leader
	// only used for api to find the current leader
	// api may need multiple hops to find the leader
	HopToCurrentLeaderId uint32

	// the channel for triggering the log committing process
	CommitChan chan struct{}
	// the channel for triggering sending AppendEntries to all followers
	AppendEntryChan chan struct{}
	// the stop channel for main goroutine to exit
	StopChan chan struct{}


	// the voted peer id
	VotedPeer uint32
	// the election start time
	ElectionRestartTime time.Time
	
	// the on-disk LogEntryList writer
	DiskLogEntry *os.File
}

func NewNodeContext(currentTerm uint64, commitIndex uint64,
	lastAppliedIndex uint64, lastBackupIndex uint64, nodeState int,
	hopToCurrentLeaderId uint32, commitChan chan struct{},
	appendEntryChan chan struct{}, stopChan chan struct{}, votedPeer uint32,
	electionRestartTime time.Time, diskLogEntry *os.File) *NodeContext {
	return &NodeContext{
		CurrentTerm:          currentTerm,
		CommitIndex:          commitIndex,
		LastAppliedIndex:     lastAppliedIndex,
		LastBackupIndex:      lastBackupIndex,
		NodeState:            nodeState,
		HopToCurrentLeaderId: hopToCurrentLeaderId,
		CommitChan:           commitChan,
		AppendEntryChan:      appendEntryChan,
		StopChan:             stopChan,
		VotedPeer:            votedPeer,
		ElectionRestartTime:  electionRestartTime,
		DiskLogEntry:         diskLogEntry,
	}
}

// used when starting the raft cluster
func NewStartNodeContext(config *NodeConfig) (*NodeContext, error) {
	// read config file as []byte
	fileData, err := os.OpenFile(config.Storage.EntryStoragePath, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil{
		return nil, err
	}
	return NewNodeContext(1,0, 0, 0,
		 Dead, 0, make(chan struct{}, 1), make(chan struct{}, 1),
		make(chan struct{}, 1), 0, time.Now(), fileData), nil
}

func (nc *NodeContext) TriggerAEChannel() {
	if len(nc.AppendEntryChan) == 0 {
		nc.AppendEntryChan <- struct{}{}
	}
}

func (nc *NodeContext) TriggerCommitChannel()  {
	if len(nc.CommitChan) == 0 {
		nc.CommitChan <- struct{}{}
	}
}


