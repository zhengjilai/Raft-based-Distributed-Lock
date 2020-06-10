// the interchange before dlock command is appended into LogMemory
package node

import (
	"github.com/dlock_raft/storage"
)

// the struct for processing dlock requests, before appending valid LogEntry to LogMemory
type DlockInterchange struct {

	// the NodeRef Object
	NodeRef *Node

	// the term for this DlockInterchange
	// note that only within the same term, this struct can be valid
	LeaderTerm uint64

	// the map for pending dlock
	// key: lockName, unique, format string
	PendingAcquire map[string]*DlockVolatileAcquirement
}

// this constructor can only be invoked when the node first become a leader
// should enter with mutex
func NewDlockInterchange(nodeRef *Node) *DlockInterchange {

	return &DlockInterchange{
		NodeRef: nodeRef,
		LeaderTerm: nodeRef.NodeContextInstance.CurrentTerm,
		PendingAcquire: make(map[string]*DlockVolatileAcquirement),
	}
}

// this initializer and only be invoked when the node first become a leader
// should enter with mutex
func (di *DlockInterchange) InitFromDLockStateMap(timestamp int64) error {

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()

	statemap := di.NodeRef.StateMapDLock
	currentTerm := di.NodeRef.NodeContextInstance.CurrentTerm
	appendNumber := uint64(0)
	var logEntries []*storage.LogEntry

	// inspect the state of all locks
	for lockName := range statemap.StateMap {
		// get current dlock info from statemap
		lockState, err := statemap.QuerySpecificState(lockName)
		if err != nil {
			return err
		}
		lockStateDecoded, ok := lockState.(*storage.DlockState)
		if !ok {
			return storage.InMemoryStateMapDLockInfoDecodeError
		}

		// if the lock is expired, but still owned by someone
		if lockStateDecoded.Owner != "" &&
			(lockStateDecoded.Timestamp + lockStateDecoded.Expire < timestamp) {
			// construct a new LogEntry, in order to release the dlock
			currentLogEntry, err := di.constructLogEntryToReleaseDlock(lockStateDecoded.LockNonce + 1,
				lockStateDecoded.LockName, "", timestamp, 0,
				currentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + appendNumber + 1)
			if err != nil {
				return err
			}
			// append current entry
			logEntries = append(logEntries, currentLogEntry)
			appendNumber += 1
		}
	}
	// append those logEntries for dlock release into LogEntry
	err := di.NodeRef.LogEntryInMemory.InsertValidEntryList(logEntries)
	if err != nil {
		return err
	}
	di.NodeRef.NodeContextInstance.TriggerAEChannel()
	return nil
}

// construct a LogEntry in order to release an expired dlock
// should enter with mutex
func (di *DlockInterchange) constructLogEntryToReleaseDlock(lockNonce uint32, lockName string, newOwner string,
	timestamp int64, expire int64, currentTerm uint64, entryIndex uint64) (*storage.LogEntry, error) {

	// construct the target logEntry
	command, err := storage.NewCommandDLock(lockNonce, lockName, newOwner, timestamp, expire)
	if err != nil {
		return nil, err
	}
	logEntry, err := storage.NewLogEntry(currentTerm, entryIndex, command)
	if err != nil {
		return nil, err
	}
	return logEntry, nil
}

// acquire a dlock
func (di *DlockInterchange) AcquireDLock(lockName string){

}