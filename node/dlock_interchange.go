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
	"unexpected error for dlock acquire")
var ReleaseDLockUnexpectedError = errors.New("dlock_raft.dlock_interchange: " +
	"unexpected error for dlock release")

const (
	ErrorReserve= iota + 100
	NoDLockExist
	TriggerReleaseLater
	AlreadyReleased
	TriggerReleaseSuccess
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
	err := di.releaseExpiredDLocks(timestamp)
	if err != nil {
		return err
	}

	return nil
}

// release all expired dlocks in statemap, acording to the current timestamp
// new LogEntries for releasing dlock (owner = "") will be appended to LogMemory
// should enter with mutex
func (di *DlockInterchange) releaseExpiredDLocks(timestamp int64) error {

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

		// construct a pending acquire if not existing, often used for interchange init
		pendingAcq, ok2 := di.PendingAcquire[lockStateDecoded.LockName]
		if !ok2 {
			di.PendingAcquire[lockStateDecoded.LockName] = NewDlockVolatileAcquirement(
				lockStateDecoded.LockName, lockStateDecoded.LockNonce)
		}
		err = pendingAcq.AbandonExpiredAcquirement(timestamp)
		if err != nil {
			return err
		}

		// refresh dlock only if it is expired
		if lockStateDecoded.Timestamp + lockStateDecoded.Expire < timestamp {

			if pendingAcq.LastAppendedNonce > lockState.(*storage.DlockState).LockNonce {
				// meaning there is a LogEntry appended but not committed, then skip this dlock
				continue
			}

			// get last valid acquirement
			command, err := pendingAcq.PopFirstValidAcquirement(timestamp)
			if err != nil {
				return err
			}

			if command == nil {
				// command == nil means no valid acquirement is available
				// refresh dlock only if the lock is still owned by someone (then release it)
				if lockStateDecoded.Owner != "" {
					// construct a new LogEntry, in order to release the dlock
					currentLogEntry, err := di.constructLogEntryByItems(lockStateDecoded.LockNonce + 1,
						lockStateDecoded.LockName, "", timestamp, 0,
						currentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + appendNumber + 1)
					if err != nil {
						return err
					}
					// append current entry
					logEntries = append(logEntries, currentLogEntry)
					appendNumber += 1
				}
			} else {
				// if there is a popped command, then insert the new logEntry in LogMemory
				logEntry, err := storage.NewLogEntry(
					currentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + appendNumber + 1, command)
				if err != nil {
					return err
				}
				logEntries = append(logEntries, logEntry)
				appendNumber += 1
			}
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
func (di *DlockInterchange) constructLogEntryByItems(lockNonce uint32, lockName string, newOwner string,
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
// will pop an acquirement if necessary
func (di *DlockInterchange) RefreshDLockByPendingAcquirement(dlockName string, timestamp int64) error {

	dlockAcq, ok := di.PendingAcquire[dlockName]
	// do nothing if no such dlock exists
	if !ok {
		di.NodeRef.NodeLogger.Debugf("When refreshing Dlock %s, it does not exist, thus do nothing\n", dlockName)
		return nil
	}

	// here we make sure that at most one LogEntry can be in LogMemory but not committed
	currentLockState, err := di.NodeRef.StateMapDLock.QuerySpecificState(dlockName)
	if err != nil {
		return err
	}
	if dlockAcq.LastAppendedNonce > currentLockState.(*storage.DlockState).LockNonce {
		di.NodeRef.NodeLogger.Debugf("Stop refreshing a Dlock %s at nonce %d, " +
			"as a LogEntry is appended but not committed.\n", dlockName, dlockAcq.LastAppendedNonce)
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
	di.NodeRef.NodeContextInstance.TriggerAEChannel()
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
			di.NodeRef.NodeContextInstance.TriggerAEChannel()
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
		// note that according to our design, if dlock state is not nil, there should be an acquirement handler
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
		di.NodeRef.NodeContextInstance.TriggerAEChannel()
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


// refresh a dlock acquirement specified by sequence number
// should enter with mutex
// note that sequence is returned by AcquireDlock, and already expired acquirement can never be refreshed
func (di *DlockInterchange) RefreshAcquirementBySequence(lockName string, sequence uint32, timestamp int64)(bool, error){

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	// refresh acquirement before acquiring a dlock
	err := di.RefreshDLockByPendingAcquirement(lockName, timestamp)
	if err != nil {
		return false, err
	}

	dlockAcq, ok := di.PendingAcquire[lockName]
	// do nothing if no such dlock exists
	if !ok {
		di.NodeRef.NodeLogger.Debugf("When refreshing DLock acquirement %s," +
			" it does not exist, thus do nothing\n", lockName)
		return false, nil
	}

	err = dlockAcq.RefreshAcquirement(sequence, timestamp)
	if err != nil {
		return false, err
	} else {
		return true, nil
	}
}


// query the specific dlock state, currently only pending acquirement number
// should enter with mutex
// query acquirement information only if leaderTag == true
func (di *DlockInterchange) QueryDLockAcquirementInfo(lockName string) int32 {

	pendingAcq, ok := di.PendingAcquire[lockName]
	if !ok {
		return -1
	}
	return int32(pendingAcq.lastAssignedAcquirement - pendingAcq.lastProcessedAcquirement)
}

// release a dlock
// should enter with mutex
// (NoDLockExist, nil) meaning the dlock does not exist in statemap (do not need to release)
// (TriggerReleaseLater, nil) meaning a LogEntry is appended but not committed, should come after some time
// (AlreadyReleased, nil) meaning the lock has already been released
// (TriggerReleaseSuccess, nil) meaning the lock release LogEntry is appended to LogMemory
func (di *DlockInterchange) ReleaseDLock(lockName string, applicant string, timestamp int64) (uint, error){

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	// refresh acquirement before acquiring a dlock
	err := di.RefreshDLockByPendingAcquirement(lockName, timestamp)
	if err != nil {
		return ErrorReserve, err
	}

	dlockAcq, okPA := di.PendingAcquire[lockName]

	// get current dlock info from statemap
	lockState, err := di.NodeRef.StateMapDLock.QuerySpecificState(lockName)

	// meaning no such dlock committed in statemap
	if err == storage.InMemoryStateMapDLockFetchError {
		// also no volatile dlock pending acquirement exists, then dlock does not exist
		if okPA == true && dlockAcq.LastAppendedNonce > 0{
			return TriggerReleaseLater, nil
		} else {
			return NoDLockExist, nil
		}
	} else if err != nil{
		return ErrorReserve, err
	}

	// now get the decoded lock state
	lockStateDecoded, ok := lockState.(*storage.DlockState)
	if !ok {
		return ErrorReserve, storage.InMemoryStateMapDLockInfoDecodeError
	}

	// if the lock has already been released (or actually expired)
	if lockStateDecoded.Owner != applicant {
		return AlreadyReleased, nil
	}
	if okPA == true && dlockAcq.LastAppendedNonce > lockStateDecoded.LockNonce {
		return TriggerReleaseLater, nil
	} else if okPA == false {
		return ErrorReserve, EmptyDLockVolatileAcquirementError
	} else {
		currentLogEntry, err := di.constructLogEntryByItems(lockStateDecoded.LockNonce + 1,
			lockStateDecoded.LockName, "", timestamp, 0,
			di.NodeRef.NodeContextInstance.CurrentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + 1)
		if err != nil {
			return ErrorReserve, err
		}
		err = di.NodeRef.LogEntryInMemory.InsertLogEntry(currentLogEntry)
		if err != nil {
			return ErrorReserve, err
		}
	}
	return ErrorReserve, ReleaseDLockUnexpectedError
}

