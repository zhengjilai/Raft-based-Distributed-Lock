// the interchange before dlock command is appended into LogMemory
package node

// the struct for processing dlock requests, before appending valid LogEntry to LogMemory
type DlockInterchange struct {

	// the NodeRef Object
	NodeRef *Node

	// the term for this DlockInterchange
	// note that only within the same term, this struct can be valid
	LeaderTerm uint64

	// the map for pending dlock
	// key: lockName, unique, format string
	PendingAcquire map[string]map[uint32]*DlockVolatileAcquirement
}

