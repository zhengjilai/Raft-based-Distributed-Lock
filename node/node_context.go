// an implementation for NodeContext
// all information in NodeContext are dynamic
// often related to the current node state, and is changing rapidly
package node

import "time"

const (
	Dead = iota
	Leader
	Candidate
	Follower
	Unknown
)

// NodeContext is the concrete implementation of NodeContext.
type NodeContext struct {
	
	// the last index in local LogMemory
	CurrentIndex uint64
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
	
	// the current leader
	CurrentLeader *PeerNode

	// the channel for triggering the log committing process
	CommitChan chan struct{}
	// the channel for triggering sending AppendEntries to all followers
	AppendEntryChan chan struct{}


	// the voted peer id
	VotedPeer uint32
	// the election start time
	electionRestartTime time.Time

}

func NewNodeContext(currentIndex uint64, currentTerm uint64, commitIndex uint64,
	lastAppliedIndex uint64, lastBackupIndex uint64,
	nodeState int, currentLeader *PeerNode) *NodeContext {
	return &NodeContext{CurrentIndex: currentIndex,
					CurrentTerm: currentTerm,
					CommitIndex: commitIndex,
					LastAppliedIndex: lastAppliedIndex,
					LastBackupIndex: lastBackupIndex,
					NodeState: nodeState,
					CurrentLeader: currentLeader}
}

// used when starting the raft cluster
func NewStartNodeContext() *NodeContext {
	return NewNodeContext(0,0, 0,
		0, 0, Dead, nil)
}


