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

// This constructor can only be invoked when the node first become a leader
// should enter with mutex
func NewDlockInterchange(nodeRef *Node) *DlockInterchange {

	return &DlockInterchange{
		NodeRef: nodeRef,
		LeaderTerm: nodeRef.NodeContextInstance.CurrentTerm,
		PendingAcquire: make(map[string]*DlockVolatileAcquirement),
	}
}

// This initializer and only be invoked when the node first become a leader
// should enter with mutex
func (di *DlockInterchange) InitFromDLockStateMap(timestamp int64) error {

	di.NodeRef.NodeLogger.Debugf("Init a DlockInterchange from statemap, term %d", di.LeaderTerm)
	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()
	err := di.scanDLockStateMap(timestamp)
	if err != nil {
		return err
	}
	return nil
}

// Release all expired dlocks in statemap, according to the current timestamp
// Often used when initializing a DLockInterchange, or scan DLockStatemap periodically
// should enter with mutex
func (di *DlockInterchange) scanDLockStateMap(timestamp int64) error {

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

		// construct a DLockVolatileAcquirement if not existing, often used for interchange init
		pendingAcq, ok2 := di.PendingAcquire[lockStateDecoded.LockName]
		if !ok2 {
			di.PendingAcquire[lockStateDecoded.LockName] = NewDlockVolatileAcquirement(
				lockStateDecoded.LockName, lockStateDecoded.LockNonce)
			pendingAcq = di.PendingAcquire[lockStateDecoded.LockName]
			di.NodeRef.NodeLogger.Debugf("Init a DLockVolatileAcquirement for dlock %s at term %d, with nonce %d",
				lockName, di.LeaderTerm, lockStateDecoded.LockNonce)
		}
		err = pendingAcq.AbandonExpiredAcquirement(timestamp)
		if err != nil {
			di.NodeRef.NodeLogger.Debugf("Begin to abandon all expired acquirement for dlock %s at term %d",
				lockName, di.LeaderTerm)
			return err
		}

		// if there is a LogEntry appended but not committed, then skip this dlock
		currentNonce := lockState.(*storage.DlockState).LockNonce
		if pendingAcq.LastAppendedNonce > currentNonce {
			di.NodeRef.NodeLogger.Debugf("Stop when processing a Dlock %s at nonce %d, " +
				"as a LogEntry with nonce %d is appended but not committed.",
				lockName, pendingAcq.LastAppendedNonce, currentNonce)
			continue
		}

		// refresh dlock only if it is expired
		if lockStateDecoded.Timestamp + lockStateDecoded.Expire < timestamp {

			if lockStateDecoded.Owner != "" {
				// construct a new LogEntry, in order to release the dlock
				// remember that for a LogEntry for pure dlock release, expire is set to be 0
				currentLogEntry, err := di.constructLogEntryByItems(lockStateDecoded.LockNonce + 1,
					lockStateDecoded.LockName, "", timestamp, 0,
					currentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + appendNumber + 1)
				if err != nil {
					return err
				}
				// append current entry
				logEntries = append(logEntries, currentLogEntry)
				appendNumber += 1
				di.NodeRef.NodeLogger.Debugf("Begin to release expired dlock %s in statemap, nonce %d, original owner %s, term %d",
					lockStateDecoded.LockName, lockStateDecoded.LockNonce, lockStateDecoded.Owner, di.LeaderTerm)
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
		// don't forget to increment lastAppendedNonce
		di.PendingAcquire[currentDLockInfo.LockName].LastAppendedNonce += 1
	}
	di.NodeRef.NodeLogger.Debugf("Expired dlocks are released in term %d, totally %d",
		di.LeaderTerm, len(logEntries))
	// trigger appendEntry
	di.NodeRef.NodeContextInstance.TriggerAEChannel()
	return nil
}


// This function will construct a LogEntry in order to release an expired dlock
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


// Refresh a specific dlock
// 1. This function may pop an acquirement if necessary, but the acquirement should be owned by the specific client
// 2. Expire LogEntry will not be created here (LogEntry for directly releasing dlock, see scanDLockStateMap)
// This design is because we can save a LogEntry by merging release and acquirement
// 3. The returned nonce is the nonce of appended LogEntry, often used when acquiring a dlock
// should enter with mutex
func (di *DlockInterchange) refreshSpecificDLock(dlockName string, clientID string, timestamp int64) (uint32, error) {

	dlockAcq, ok := di.PendingAcquire[dlockName]
	// do nothing if no such dlock exists
	if !ok {
		di.NodeRef.NodeLogger.Debugf("When refreshing Dlock %s, it does not exist, thus do nothing", dlockName)
		return 0, EmptyDLockVolatileAcquirementError
	}

	// here we make sure that at most one LogEntry can be in LogMemory but not committed
	currentLockState, err := di.NodeRef.StateMapDLock.QuerySpecificState(dlockName)
	if err != nil {
		return 0, err
	}
	decodedLockState := currentLockState.(*storage.DlockState)

	// current dlock expire check, if the specific dlock has not expired, do nothing
	if decodedLockState.Timestamp + decodedLockState.Expire > timestamp {
		return 0, nil
	}
	// also check if there is a LogEntry appended but not committed
	if dlockAcq.LastAppendedNonce > decodedLockState.LockNonce {
		di.NodeRef.NodeLogger.Debugf("Stop refreshing a Dlock %s at nonce %d, " +
			"as a LogEntry is appended but not committed.", dlockName, dlockAcq.LastAppendedNonce)
		return 0, nil
	}

	// get last valid acquirement
	command, err := dlockAcq.FetchFirstValidAcquirement(timestamp, false)
	if err != nil {
		return 0, err
	}

	if command == nil {
		// command == nil means no valid acquirement is available, then do nothing
		di.NodeRef.NodeLogger.Debugf("No available pending acquirement for dlock %s at nonce %d",
			dlockName, dlockAcq.LastAppendedNonce)
		return 0, nil
	} else {
		dlockInfo, err := command.GetAsDLockInfo()
		if err != nil {
			return 0, nil
		}
		// only process the command with the same invoker as clientId
		if dlockInfo.NewOwner != clientID {
			return 0, nil
		}
	}

	// now command must exist, and deal with the same client, then pop the last command
	command, err = dlockAcq.FetchFirstValidAcquirement(timestamp, true)
	if err != nil {
		return 0, err
	}
	// has a popped command, then insert the new logEntry
	logEntry, err := storage.NewLogEntry(
		di.NodeRef.NodeContextInstance.CurrentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + 1, command)
	if err != nil {
		return 0, err
	}
	err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
	if err != nil {
		return 0, err
	}

	// refresh LastAppendedNonce if LogMemory is refreshed with a logEntry
	di.PendingAcquire[dlockName].LastAppendedNonce += 1
	di.NodeRef.NodeContextInstance.TriggerAEChannel()
	di.NodeRef.NodeLogger.Debugf("Dlock %s with nonce %d is appended to LogMemory.",
		dlockName, di.PendingAcquire[dlockName].LastAppendedNonce)
	return di.PendingAcquire[dlockName].LastAppendedNonce, nil
}


// The interface for acquiring a dlock
// should enter with mutex
// Return: (sequence, nonce, err)
// sequence != 0 : acquirement succeeded by appending it to pending list, then nonce is nonsense
// sequence == 0 : acquirement succeeded by appending a LogEntry to LogMemory,
// then nonce is the nonce of appended LogEntry
func (di *DlockInterchange) AcquireDLock(lockName string,
	timestamp int64, command *storage.CommandDLock)(uint32, uint32, error) {

	di.NodeRef.NodeLogger.Debugf("Begin to acquire dlock %s in term %d", lockName, di.LeaderTerm)

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()

	dlockInfo, err := command.GetAsDLockInfo()
	if err != nil {
		return 0, 0, err
	}

	// get current dlock state
	currentDLockState, err2 := di.NodeRef.StateMapDLock.QuerySpecificState(lockName)
	if err2 != storage.InMemoryStateMapDLockFetchError && err2 != nil {
		return 0, 0, err
	}

	// case1: dlock does not exist in Statemap
	// operation: append a dlock LogEntry, as well as create/refresh a volatile acquirement pending list
	if err2 == storage.InMemoryStateMapDLockFetchError {
		di.NodeRef.NodeLogger.Debugf("AcquireDLock case1: dlock %s does not exist in Statemap.", lockName)
		// update lockNonce
		dlockInfo.LockNonce = 1
		dlockInfo.Timestamp = timestamp
		err2 := command.SetAsDLockInfo(dlockInfo)
		if err2 != nil {
			return 0, 0, err2
		}
		di.NodeRef.NodeLogger.Debugf("DLock info of AcquireDLock command in case1: %+v", dlockInfo)
		pendingAcq, ok := di.PendingAcquire[lockName]
		if !ok {
			// if no existing Volatile Acquirement, new one
			di.NodeRef.NodeLogger.Debugf("AcquireDLock case1-branch1: " +
				"dlock %s has no existing Volatile Acquirement.", lockName)
			di.PendingAcquire[lockName] = NewDlockVolatileAcquirement(lockName, 0)
			// directly insert LogEntry to LogMemory
			logEntry, err := storage.NewLogEntry(di.NodeRef.NodeContextInstance.CurrentTerm,
				di.NodeRef.LogEntryInMemory.MaximumIndex() + 1, command)
			if err != nil {
				return 0, 0, err
			}
			err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
			if err != nil {
				return 0, 0, err
			}
			di.PendingAcquire[lockName].LastAppendedNonce += 1
			di.NodeRef.NodeContextInstance.TriggerAEChannel()
			di.NodeRef.NodeLogger.Debugf("Acquire process finishes by creating a new DLock %s.", lockName)
			return 0, di.PendingAcquire[lockName].LastAppendedNonce, nil
		} else {
			// if there is already a volatile acquirement, just insert the acquirement
			// note that in this case lastAppendedNonce must be 1
			// as the implicit nonce for lock is 0 (no dlock entry committed)
			di.NodeRef.NodeLogger.Debugf("AcquireDLock case1-branch2: " +
				"dlock %s has an existing volatile acquirement.", lockName)
			sequence, err := pendingAcq.InsertNewAcquirement(command, timestamp,
				int64(time.Duration(di.NodeRef.NodeConfigInstance.Parameters.AcquirementExpire) * time.Millisecond))
			if err != nil {
				return 0, 0, err
			}
			di.NodeRef.NodeLogger.Debugf("Acquire process finishes by inserting a pending DLock acquirement," +
				" at sequence %d.", sequence)
			return sequence, 0, nil
		}
	}

	currentDLockStateDecoded, ok := currentDLockState.(*storage.DlockState)
	pendingAcq, ok := di.PendingAcquire[lockName]
	if !ok {
		// note that according to our design, if dlock state is not nil, there should be an acquirement handler
		return 0, 0, EmptyDLockVolatileAcquirementError
	}
	di.NodeRef.NodeLogger.Debugf("AcquireDLock dlock %s has current dlock state %+v", lockName, currentDLockStateDecoded)

	if pendingAcq.LastAppendedNonce == currentDLockStateDecoded.LockNonce &&
		pendingAcq.lastProcessedAcquirement == pendingAcq.lastAssignedAcquirement &&
		currentDLockStateDecoded.Timestamp + currentDLockStateDecoded.Expire < timestamp {

		// case2: dlock exists in state map
		// besides, we have no pending acquirement, and also no appending but uncommitted logEntry,
		// also, the current dlock is expired
		// operation: directly append a LogEntry

		di.NodeRef.NodeLogger.Debugf("AcquireDLock case2: " +
			"begin to directly append a LogEntry for Dlock %s to LogMemory.", lockName)

		// update lockNonce
		dlockInfo.LockNonce = currentDLockStateDecoded.LockNonce + 1
		dlockInfo.Timestamp = timestamp
		err2 := command.SetAsDLockInfo(dlockInfo)
		if err2 != nil {
			return 0, 0, err2
		}
		// construct logEntry and Insert it
		logEntry, err := storage.NewLogEntry(di.NodeRef.NodeContextInstance.CurrentTerm,
			di.NodeRef.LogEntryInMemory.MaximumIndex()+1, command)
		if err != nil {
			return 0, 0, err
		}
		err = di.NodeRef.LogEntryInMemory.InsertLogEntry(logEntry)
		if err != nil {
			return 0, 0, err
		}
		// don't forget to refresh last appended nonce
		pendingAcq.LastAppendedNonce += 1
		// trigger append entries
		di.NodeRef.NodeContextInstance.TriggerAEChannel()
		di.NodeRef.NodeLogger.Debugf("Acquire process finishes by directly " +
			"appending a LogEntry to LogMemory (nobody racing for DLock %s).", lockName)
		return 0, dlockInfo.LockNonce, nil

	} else if pendingAcq.LastAppendedNonce > currentDLockStateDecoded.LockNonce ||
		pendingAcq.lastProcessedAcquirement < pendingAcq.lastAssignedAcquirement ||
		currentDLockStateDecoded.Timestamp + currentDLockStateDecoded.Expire >= timestamp {

		// case3: dlock exists in state map
		// but we still have at least one processing acquirement, or an appending but uncommitted logEntry,
		// or the current dlock is not expired
		// operation: append the current command to volatile acquirement list, then fresh the acquirement list

		di.NodeRef.NodeLogger.Debugf("AcquireDLock case3: " +
			"begin to adding an acquirement for Dlock %s to pending list", lockName)

		// note that command will refreshed (LockNonce and Timestamp) when popped from acquirement list
		sequence, err := pendingAcq.InsertNewAcquirement(command, timestamp,
			int64(time.Duration(di.NodeRef.NodeConfigInstance.Parameters.AcquirementExpire)*time.Millisecond))
		if err != nil {
			return 0, 0, err
		}

		// refresh a specific dlock
		nonce, err := di.refreshSpecificDLock(lockName, dlockInfo.NewOwner, timestamp)
		if err != nil {
			return 0, 0, err
		}

		di.NodeRef.NodeLogger.Debugf("Acquire process finishes by inserting a pending DLock acquirement," +
			" at sequence %d, nonce %d", sequence, nonce)
		return sequence, 0, nil
	}

	di.NodeRef.NodeLogger.Debugf("Unexpected situation occurs when acquiring dlock %s, " +
		"current nonce %d, last appended nonce %d, last processed acq %d, last assigned acq %d",
		lockName, currentDLockStateDecoded.LockNonce, pendingAcq.LastAppendedNonce,
		pendingAcq.lastProcessedAcquirement, pendingAcq.lastAssignedAcquirement)
	return 0, 0, AcquireDLockUnexpectedError
}


// refresh a dlock acquirement specified by sequence number
// should enter with mutex
// note that sequence is returned by AcquireDlock, and already expired acquirement can never be refreshed
func (di *DlockInterchange) RefreshAcquirementBySequence(lockName string, sequence uint32, timestamp int64)(bool, error){

	di.NodeRef.NodeLogger.Debugf("Begin to refresh dlock %s acquirement seq %d in term %d",
		lockName, sequence, di.LeaderTerm)

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()

	dlockAcq, ok := di.PendingAcquire[lockName]
	// do nothing if no such dLock exists
	if !ok {
		di.NodeRef.NodeLogger.Debugf("When refreshing DLock acquirement %s at sequence %d," +
			" it does not exist, thus do nothing", lockName, sequence)
		return false, nil
	}
	// abandon expired acquirement before refresh them
	err := dlockAcq.AbandonExpiredAcquirement(timestamp)
	if err != nil {
		return false, err
	}

	err = dlockAcq.RefreshAcquirement(sequence, timestamp)
	if err == VolatileAcquirementExpireError {
		di.NodeRef.NodeLogger.Debugf("When refreshing DLock acquirement %s at sequence %d," +
			" it has already expired, thus do nothing", lockName, sequence)
		return false, VolatileAcquirementExpireError
	} else if err != nil {
		di.NodeRef.NodeLogger.Debugf("When refreshing DLock acquirement %s at sequence %d," +
			" it has a sequence error, thus do nothing, error: %s", lockName, sequence, err)
		return false, err
	} else {
		di.NodeRef.NodeLogger.Debugf("Refresh acquirement for DLock %s by sequence %d succeeded.",
			lockName, sequence)
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

// The interface for releasing a dlock
// should enter with mutex
// (NoDLockExist, 0, nil) meaning the dlock does not exist in statemap (do not need to release)
// (TriggerReleaseLater, 0, nil) meaning a LogEntry is appended but not committed, should come after some time
// (AlreadyReleased, 0, nil) meaning the lock has already been released
// (TriggerReleaseSuccess, nonce, nil) meaning the lock release LogEntry is appended to LogMemory, nonce is also returned
func (di *DlockInterchange) ReleaseDLock(lockName string, applicant string, timestamp int64) (uint, uint32, error){

	di.NodeRef.NodeLogger.Debugf("Begin to release dlock %s in term %d",
		lockName, di.LeaderTerm)

	// make sure the current MemoryStateMap is up-to-date
	di.NodeRef.commitProcedure()

	dlockAcq, okPA := di.PendingAcquire[lockName]

	// get current dlock info from statemap
	lockState, err := di.NodeRef.StateMapDLock.QuerySpecificState(lockName)

	// meaning no such dlock committed in statemap
	if err == storage.InMemoryStateMapDLockFetchError {
		// also no volatile dlock pending acquirement exists, then dlock does not exist
		if okPA == true && dlockAcq.LastAppendedNonce > 0{
			di.NodeRef.NodeLogger.Debugf("Trigger release later, " +
				"as some LogEntry for DLock %s is processing.", lockName)
			return TriggerReleaseLater, 0, nil
		} else {
			di.NodeRef.NodeLogger.Debugf("No DLock %s exists.", lockName)
			return NoDLockExist, 0, nil
		}
	} else if err != nil{
		return ErrorReserve, 0, err
	}

	// now get the decoded lock state
	lockStateDecoded, ok := lockState.(*storage.DlockState)
	if !ok {
		return ErrorReserve, 0, storage.InMemoryStateMapDLockInfoDecodeError
	}

	// if the lock has already been released (or actually expired)
	if lockStateDecoded.Owner != applicant {
		di.NodeRef.NodeLogger.Debugf("DLock %s already released or expired, current owner %s.",
			lockName, lockStateDecoded.Owner)
		return AlreadyReleased, 0, nil
	}
	if okPA == true && dlockAcq.LastAppendedNonce > lockStateDecoded.LockNonce {
		di.NodeRef.NodeLogger.Debugf("Trigger release later, " +
			"as some LogEntry for DLock %s is processing.", lockName)
		return TriggerReleaseLater, 0,  nil
	} else if okPA == false {
		return ErrorReserve, 0, EmptyDLockVolatileAcquirementError
	} else {
		currentLogEntry, err := di.constructLogEntryByItems(lockStateDecoded.LockNonce + 1,
			lockStateDecoded.LockName, "", timestamp, 0,
			di.NodeRef.NodeContextInstance.CurrentTerm, di.NodeRef.LogEntryInMemory.MaximumIndex() + 1)
		if err != nil {
			return ErrorReserve, 0, err
		}
		err = di.NodeRef.LogEntryInMemory.InsertLogEntry(currentLogEntry)
		if err != nil {
			return ErrorReserve, 0, err
		}
		// don't forget to refresh last appended nonce
		dlockAcq.LastAppendedNonce += 1
		// trigger append entries
		di.NodeRef.NodeContextInstance.TriggerAEChannel()
		di.NodeRef.NodeLogger.Debugf("Trigger release of dlock %s succeeded by appending an LogEntry.", lockName)
		return TriggerReleaseSuccess, lockStateDecoded.LockNonce + 1, nil
	}
}

// The periodically running goroutine during the whole leader term
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
			err := di.scanDLockStateMap(now)
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