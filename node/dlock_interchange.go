// the interchange before dlock command is appended into LogMemory
package node

import (
	"errors"
	"github.com/dlock_raft/storage"
	"time"
)

var EmptyDLockVolatileAcquirementError = errors.New("dlock_raft.dlock_interchange: " +
	"unexpected empty dlock volatile acquirement")
var AcquireDLockUnexpectedError = errors.New("dlock_raft.dlock_interchange: " +
	"last appended nonce smaller than committed nonce")

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
		di.PendingAcquire[lockStateDecoded.LockName] = NewDlockVolatileAcquirement(
			lockStateDecoded.LockName, lockStateDecoded.LockNonce)

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
	// refresh last appended nonce for every appended LogEntry
	for _, logEntry := range logEntries {
		currentCommand := storage.NewCommandFromRaw("DLock", logEntry.Entry.GetCommandContent())
		currentDLockInfo, err := currentCommand.(*storage.CommandDLock).GetAsDLockInfo()
		if err != nil {
			return err
		}
		di.PendingAcquire[currentDLockInfo.LockName].LastAppendedNonce += 1
	}
	// trigger appendEntry
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

// refresh a specific dlock
func (di *DlockInterchange) RefreshDLockByPendingAcquirement(dlockName string, timestamp int64) error {

	dlockAcq, ok := di.PendingAcquire[dlockName]
	// do nothing if no such dlock exists
	if !ok {
		di.NodeRef.NodeLogger.Debugf("When refreshing Dlock %s, it does not exist.\n", dlockName)
		return nil
	}
	// here we make sure that at most one LogEntry can be in LogMemory but not committed
	currentLockState, err := di.NodeRef.StateMapDLock.QuerySpecificState(dlockName)
	if err != nil {
		return err
	}
	if dlockAcq.LastAppendedNonce > currentLockState.(*storage.DlockState).LockNonce {
		di.NodeRef.NodeLogger.Debugf("Stop refreshing a Dlock %s at nonce %d, " +
			"as some LogEntry is appended but not committed.\n", dlockName, dlockAcq.LastAppendedNonce)
		return nil
	}

	// get last valid acquirement
	command, err := dlockAcq.PopFirstValidAcquirement(timestamp)
	if err != nil {
		return err
	}

	if command == nil {
		// command == nil means no valid acquirement is available, then do nothing
		di.NodeRef.NodeLogger.Debugf("No available pending acquirement for dlock %s at nonce %d",
			dlockName, dlockAcq.LastAppendedNonce)
		return nil
	} else {
		// has a popped command, then insert the new logEntry
		logEntry, err := storage.NewLogEntry(
			di.NodeRef.NodeContextInstance.CurrentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + 1, command)
		if err != nil {
			return err
		}
		err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
		if err != nil {
			return err
		}
	}
	// refresh LastAppendedNonce if LogMemory is refreshed with a logEntry
	di.PendingAcquire[dlockName].LastAppendedNonce += 1
	di.NodeRef.NodeLogger.Infof("Dlock %s with nonce %d is appended to LogMemory.\n",
		dlockName, di.PendingAcquire[dlockName].LastAppendedNonce)
	return nil
}

// acquire a dlock
// should enter with mutex
func (di *DlockInterchange) AcquireDLock(lockName string,
	timestamp int64, command *storage.CommandDLock)(bool, uint32, error) {

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	// refresh acquirement before acquiring a dlock
	err := di.RefreshDLockByPendingAcquirement(lockName, timestamp)
	if err != nil {
		return false, 0, err
	}
	dlockInfo, err := command.GetAsDLockInfo()
	if err != nil {
		return false, 0, err
	}

	// get current dlock state
	currentDLockState, err := di.NodeRef.StateMapDLock.QuerySpecificState(lockName)
	if err != storage.InMemoryStateMapDLockFetchError && err != nil {
		return false, 0, err
	}

	// case1: dlock does not exist in Statemap
	// operation: append a dlock LogEntry, as well as create/refresh a volatile acquirement pending list
	if err == storage.InMemoryStateMapDLockFetchError {
		// update lockNonce
		dlockInfo.LockNonce = 1
		dlockInfo.Timestamp = timestamp
		err2 := command.SetAsDLockInfo(dlockInfo)
		if err2 != nil {
			return false, 0, err2
		}
		pendingAcq, ok := di.PendingAcquire[lockName]
		if !ok {
			// if no existing Volatile Acquirement, new one
			di.PendingAcquire[lockName] = NewDlockVolatileAcquirement(lockName, 1)
			// directly insert LogEntry to LogMemory
			logEntry, err := storage.NewLogEntry(di.NodeRef.NodeContextInstance.CurrentTerm,
				di.NodeRef.LogEntryInMemory.MaximumIndex()+1, command)
			if err != nil {
				return false, 0, err
			}
			err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
			if err != nil {
				return false, 0, err
			}
			return true, 0, nil
		} else {
			// if there is already a volatile acquirement, insert the acquirement
			// note that in this case lastAppendedNonce must be 1
			// as the implicit nonce for lock is 0 (no dlock entry committed)
			sequence, err := pendingAcq.InsertNewAcquirement(command, timestamp,
				int64(time.Duration(di.NodeRef.NodeConfigInstance.Parameters.AcquirementExpire)*time.Millisecond))
			if err != nil {
				return false, 0, err
			}
			return true, sequence, nil
		}
	}

	// case2: dlock exists in state map
	// besides, we have no pending acquirement, and also no appending but uncommitted logEntry
	// operation: directly append a LogEntry

	// case3: dlock exists in state map
	// but we still have at least one pending acquirement, or an appending but uncommitted logEntry
	// operation: append the current command to volatile acquirement list
	currentDLockStateDecoded := currentDLockState.(*storage.DlockState)
	pendingAcq, ok := di.PendingAcquire[lockName]
	if !ok {
		// note that according to our design, if dlock state is not null, there should be an acquirement handler
		return false, 0, EmptyDLockVolatileAcquirementError
	}
	if pendingAcq.LastAppendedNonce == currentDLockStateDecoded.LockNonce &&
		pendingAcq.lastProcessedAcquirement == pendingAcq.lastAssignedAcquirement {
		// case2
		// update lockNonce
		dlockInfo.LockNonce = currentDLockStateDecoded.LockNonce + 1
		dlockInfo.Timestamp = timestamp
		err2 := command.SetAsDLockInfo(dlockInfo)
		if err2 != nil {
			return false, 0, err2
		}
		// construct logEntry and Insert it
		logEntry, err := storage.NewLogEntry(di.NodeRef.NodeContextInstance.CurrentTerm,
			di.NodeRef.LogEntryInMemory.MaximumIndex()+1, command)
		if err != nil {
			return false, 0, err
		}
		err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
		if err != nil {
			return false, 0, err
		}
		return true, 0, nil
	} else if pendingAcq.LastAppendedNonce > currentDLockStateDecoded.LockNonce ||
		pendingAcq.lastProcessedAcquirement < pendingAcq.lastAssignedAcquirement {
		// case 3
		// note that command will be refreshed (LockNonce and Timestamp) when popped from acquirement list
		sequence, err := pendingAcq.InsertNewAcquirement(command, timestamp,
			int64(time.Duration(di.NodeRef.NodeConfigInstance.Parameters.AcquirementExpire)*time.Millisecond))
		if err != nil {
			return false, 0, err
		}
		return true, sequence, nil
	}
	di.NodeRef.NodeLogger.Debugf("Unexpected situation occurs when acquiring dlock %s, " +
		"current nonce %d, last appended nonce %d, last processed acq %d, last assigned acq %d\n",
		lockName, currentDLockStateDecoded.LockNonce, pendingAcq.LastAppendedNonce,
		pendingAcq.lastProcessedAcquirement, pendingAcq.lastAssignedAcquirement)
	return false, 0, AcquireDLockUnexpectedError
}

