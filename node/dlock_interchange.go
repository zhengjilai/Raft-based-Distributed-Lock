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

	di.NodeRef.NodeLogger.Debugf("Init a DlockInterchange from statemap, term %d", di.LeaderTerm)
	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	err := di.refreshOrInitDLocks(timestamp)
	if err != nil {
		return err
	}
	return nil
}

// release all expired dlocks in statemap, according to the current timestamp
// should enter with mutex
func (di *DlockInterchange) refreshOrInitDLocks(timestamp int64) error {

	di.NodeRef.NodeLogger.Debugf("Begin to release all expired dlocks in statemap, term %d", di.LeaderTerm)

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
	di.NodeRef.NodeLogger.Debugf("Expired dlocks are released in term %d, totally %d",
		di.LeaderTerm, len(logEntries))
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
func (di *DlockInterchange) refreshSpecificDLock(dlockName string, timestamp int64) error {

	dlockAcq, ok := di.PendingAcquire[dlockName]
	// do nothing if no such dlock exists
	if !ok {
		di.NodeRef.NodeLogger.Debugf("When refreshing Dlock %s, it does not exist, thus do nothing", dlockName)
		return nil
	}

	// here we make sure that at most one LogEntry can be in LogMemory but not committed
	currentLockState, err := di.NodeRef.StateMapDLock.QuerySpecificState(dlockName)
	if err != nil {
		return err
	}
	if dlockAcq.LastAppendedNonce > currentLockState.(*storage.DlockState).LockNonce {
		di.NodeRef.NodeLogger.Debugf("Stop refreshing a Dlock %s at nonce %d, " +
			"as a LogEntry is appended but not committed.", dlockName, dlockAcq.LastAppendedNonce)
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
	di.NodeRef.NodeLogger.Debugf("Dlock %s with nonce %d is appended to LogMemory.",
		dlockName, di.PendingAcquire[dlockName].LastAppendedNonce)
	return nil
}


// acquire a dlock
// should enter with mutex
// return: (sequence, err)
// sequence != 0 : acquirement succeeded by appending it to pending list
// sequence == 0 : acquirement succeeded by appending a LogEntry to LogMemory
func (di *DlockInterchange) AcquireDLock(lockName string,
	timestamp int64, command *storage.CommandDLock)(uint32, error) {

	di.NodeRef.NodeLogger.Debugf("Begin to acquire dlock %s in term %d", lockName, di.LeaderTerm)

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	// refresh acquirement before acquiring a dlock
	err := di.refreshSpecificDLock(lockName, timestamp)
	if err != nil {
		return 0, err
	}
	dlockInfo, err := command.GetAsDLockInfo()
	if err != nil {
		return 0, err
	}
	di.NodeRef.NodeLogger.Debugf("DLock info of acquirement: %+v", dlockInfo)

	// get current dlock state
	currentDLockState, err2 := di.NodeRef.StateMapDLock.QuerySpecificState(lockName)
	if err2 != storage.InMemoryStateMapDLockFetchError && err2 != nil {
		return 0, err
	}

	// case1: dlock does not exist in Statemap
	// operation: append a dlock LogEntry, as well as create/refresh a volatile acquirement pending list
	if err2 == storage.InMemoryStateMapDLockFetchError {
		// update lockNonce
		dlockInfo.LockNonce = 1
		dlockInfo.Timestamp = timestamp
		err2 := command.SetAsDLockInfo(dlockInfo)
		if err2 != nil {
			return 0, err2
		}
		pendingAcq, ok := di.PendingAcquire[lockName]
		if !ok {
			// if no existing Volatile Acquirement, new one
			di.PendingAcquire[lockName] = NewDlockVolatileAcquirement(lockName, 0)
			// directly insert LogEntry to LogMemory
			logEntry, err := storage.NewLogEntry(di.NodeRef.NodeContextInstance.CurrentTerm,
				di.NodeRef.LogEntryInMemory.MaximumIndex()+1, command)
			if err != nil {
				return 0, err
			}
			err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
			if err != nil {
				return 0, err
			}
			di.PendingAcquire[lockName].LastAppendedNonce += 1
			di.NodeRef.NodeContextInstance.TriggerAEChannel()
			di.NodeRef.NodeLogger.Debugf("Acquire process finishes by creating a new DLock %s.", lockName)
			return 0, nil
		} else {
			// if there is already a volatile acquirement, insert the acquirement
			// note that in this case lastAppendedNonce must be 1
			// as the implicit nonce for lock is 0 (no dlock entry committed)
			sequence, err := pendingAcq.InsertNewAcquirement(command, timestamp,
				int64(time.Duration(di.NodeRef.NodeConfigInstance.Parameters.AcquirementExpire)*time.Millisecond))
			if err != nil {
				return 0, err
			}
			di.NodeRef.NodeLogger.Debugf("Acquire process finishes by inserting a pending DLock acquirement," +
				" at sequence %d.", sequence)
			return sequence, nil
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
		return 0, EmptyDLockVolatileAcquirementError
	}
	if pendingAcq.LastAppendedNonce == currentDLockStateDecoded.LockNonce &&
		pendingAcq.lastProcessedAcquirement == pendingAcq.lastAssignedAcquirement {
		// case2
		// update lockNonce
		dlockInfo.LockNonce = currentDLockStateDecoded.LockNonce + 1
		dlockInfo.Timestamp = timestamp
		err2 := command.SetAsDLockInfo(dlockInfo)
		if err2 != nil {
			return 0, err2
		}
		// construct logEntry and Insert it
		logEntry, err := storage.NewLogEntry(di.NodeRef.NodeContextInstance.CurrentTerm,
			di.NodeRef.LogEntryInMemory.MaximumIndex()+1, command)
		if err != nil {
			return 0, err
		}
		err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
		if err != nil {
			return 0, err
		}
		di.NodeRef.NodeContextInstance.TriggerAEChannel()
		di.NodeRef.NodeLogger.Debugf("Acquire process finishes by directly " +
			"appending a LogEntry to LogMemory (nobody racing for DLock %s).", lockName)
		return 0, nil
	} else if pendingAcq.LastAppendedNonce > currentDLockStateDecoded.LockNonce ||
		pendingAcq.lastProcessedAcquirement < pendingAcq.lastAssignedAcquirement {
		// case 3
		// note that command has already been refreshed (LockNonce and Timestamp) when popped from acquirement list
		sequence, err := pendingAcq.InsertNewAcquirement(command, timestamp,
			int64(time.Duration(di.NodeRef.NodeConfigInstance.Parameters.AcquirementExpire)*time.Millisecond))
		if err != nil {
			return 0, err
		}
		di.NodeRef.NodeLogger.Debugf("Acquire process finishes by inserting a pending DLock acquirement," +
			" at sequence %d.", sequence)
		return sequence, nil
	}
	di.NodeRef.NodeLogger.Debugf("Unexpected situation occurs when acquiring dlock %s, " +
		"current nonce %d, last appended nonce %d, last processed acq %d, last assigned acq %d",
		lockName, currentDLockStateDecoded.LockNonce, pendingAcq.LastAppendedNonce,
		pendingAcq.lastProcessedAcquirement, pendingAcq.lastAssignedAcquirement)
	return 0, AcquireDLockUnexpectedError
}


// refresh a dlock acquirement specified by sequence number
// should enter with mutex
// note that sequence is returned by AcquireDlock, and already expired acquirement can never be refreshed
func (di *DlockInterchange) RefreshAcquirementBySequence(lockName string, sequence uint32, timestamp int64)(bool, error){

	di.NodeRef.NodeLogger.Debugf("Begin to refresh dlock %s acquirement seq %d in term %d",
		lockName, sequence, di.LeaderTerm)

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	// refresh acquirement before acquiring a dLock
	err := di.refreshSpecificDLock(lockName, timestamp)
	if err != nil {
		return false, err
	}

	dlockAcq, ok := di.PendingAcquire[lockName]
	// do nothing if no such dLock exists
	if !ok {
		di.NodeRef.NodeLogger.Debugf("When refreshing DLock acquirement %s at sequence %d," +
			" it does not exist, thus do nothing", lockName, sequence)
		return false, nil
	}

	err = dlockAcq.RefreshAcquirement(sequence, timestamp)
	if err == VolatileAcquirementExpireError {
		di.NodeRef.NodeLogger.Debugf("When refreshing DLock acquirement %s at sequence %d," +
			" it has already expired, thus do nothing", lockName, sequence)
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		di.NodeRef.NodeLogger.Debugf("Refresh acquirement for DLock %s by sequence succeeded.", lockName, sequence)
		return true, nil
	}
}


// query the specific dlock state, currently only pending acquirement number
// should enter with mutex
// should query acquirement information only if node is Leader
func (di *DlockInterchange) QueryDLockAcquirementInfo(lockName string) int32 {

	pendingAcq, ok := di.PendingAcquire[lockName]
	if !ok {
		return -1
	}
	acquireNum := int32(pendingAcq.lastAssignedAcquirement - pendingAcq.lastProcessedAcquirement)
	di.NodeRef.NodeLogger.Debugf("DLock %s at term %d has %d acquirement pending.",
		lockName, di.LeaderTerm, acquireNum)
	return acquireNum
}

// release a dlock
// should enter with mutex
// (NoDLockExist, nil) meaning the dlock does not exist in statemap (do not need to release)
// (TriggerReleaseLater, nil) meaning a LogEntry is appended but not committed, should come after some time
// (AlreadyReleased, nil) meaning the lock has already been released
// (TriggerReleaseSuccess, nil) meaning the lock release LogEntry is appended to LogMemory
func (di *DlockInterchange) ReleaseDLock(lockName string, applicant string, timestamp int64) (uint, error){

	di.NodeRef.NodeLogger.Debugf("Begin to release dlock %s in term %d",
		lockName, di.LeaderTerm)

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	// refresh acquirement before acquiring a dlock
	err := di.refreshSpecificDLock(lockName, timestamp)
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
			di.NodeRef.NodeLogger.Debugf("Trigger release later, " +
				"as some LogEntry for DLock %s is processing.", lockName)
			return TriggerReleaseLater, nil
		} else {
			di.NodeRef.NodeLogger.Debugf("No DLock %s exists.", lockName)
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
		di.NodeRef.NodeLogger.Debugf("DLock %s already released or expired, current owner %s.",
			lockName, lockStateDecoded.Owner)
		return AlreadyReleased, nil
	}
	if okPA == true && dlockAcq.LastAppendedNonce > lockStateDecoded.LockNonce {
		di.NodeRef.NodeLogger.Debugf("Trigger release later, " +
			"as some LogEntry for DLock %s is processing.", lockName)
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
		di.NodeRef.NodeContextInstance.TriggerAEChannel()
		di.NodeRef.NodeLogger.Debugf("Trigger release of dlock %s succeeded by appending an LogEntry.", lockName)
		return TriggerReleaseSuccess, nil
	}
}

// the running goroutine during the whole leader term
func (di *DlockInterchange) ReleaseExpiredDLockPeriodically() {
	// ticks every fixed interval
	startLeaderTerm := di.LeaderTerm
	ticker := time.NewTicker(time.Duration(di.NodeRef.NodeConfigInstance.Parameters.PollingInterval) * time.Millisecond)
	for {
		select {
		// if semaphore for releasing expired dlock is triggered
		case <- ticker.C:
			// reserved for extreme cases, where di come nil
			if di == nil {
				return
			}
			di.NodeRef.mutex.Lock()
			// detect term change or state change
			if di.NodeRef.NodeContextInstance.CurrentTerm != startLeaderTerm{
				di.NodeRef.NodeLogger.Debugf("Term changes when periodically releasing expired dlocks," +
					" orig term %d, current term %d\n", di.LeaderTerm, di.NodeRef.NodeContextInstance.CurrentTerm)
				di.NodeRef.mutex.Unlock()
				return
			} else if di.NodeRef.NodeContextInstance.NodeState != Leader {
				di.NodeRef.NodeLogger.Debugf("Node state is not leader " +
					"when periodically releasing expired dlocks, term %d.", di.LeaderTerm)
				di.NodeRef.mutex.Unlock()
				return
			}
			now := time.Now().UnixNano()
			di.NodeRef.commitProcedure()
			err := di.refreshOrInitDLocks(now)
			if err != nil {
				di.NodeRef.NodeLogger.Debugf("Error happens when periodically releasing expired dlocks," +
					" term %d, error %s", di.LeaderTerm, err)
				di.NodeRef.mutex.Unlock()
				return
			}
			di.NodeRef.mutex.Unlock()
		}
	}
}