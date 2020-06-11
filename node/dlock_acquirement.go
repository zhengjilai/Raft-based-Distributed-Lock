package node

import (
	"errors"
	"github.com/dlock_raft/storage"
)

var VolatileAcquirementInvalidSequenceError = errors.New("dlock_raft.acquirement: " +
	"the sequence number is not valid")
var VolatileAcquirementInfoFetchingError = errors.New("dlock_raft.acquirement: " +
	"fetch the acquirement with valid sequence number fails")
var VolatileAcquirementExpireError = errors.New("dlock_raft.acquirement: " +
	"cannot refresh an acquirement if it has already expired")

// volatile information for a specific dlock request in a specific term
type DlockVolatileAcquirement struct {
	
	// the lock name
	LockName string

	// last processed acquirement sequence
	lastProcessedAcquirement uint32
	
	// last assigned acquirement sequence
	// used when new acquirement is received
	lastAssignedAcquirement uint32

	// last appended nonce (to LogMemory)
	// this field is used to prevent multiple LogEntry appended with the same Nonce
	LastAppendedNonce uint32
	
	// map: acquire seq -> actual Dlock acquirement
	PendingAcquirement map[uint32]*DlockAcquirementInfo
	
}

func NewDlockVolatileAcquirement(lockName string, lastAppendedNonce uint32) *DlockVolatileAcquirement {
	return &DlockVolatileAcquirement{
		LockName:                 lockName,
		lastAssignedAcquirement:  0,
		lastProcessedAcquirement: 0,
		LastAppendedNonce:        lastAppendedNonce,
		PendingAcquirement:       make(map[uint32]*DlockAcquirementInfo)}
}

// update an requirement with current timestamp, in case it expires
// time stamp format ns: obtained by time.Now().UnixNano()
func (dva *DlockVolatileAcquirement) RefreshAcquirement(acquireSeq uint32, timestamp int64) error {

	// first refresh pending acquirement list
	err := dva.AbandonExpiredAcquirement(timestamp)
	if err != nil {
		return err
	}

	// if the sequence number is not valid, throw error
	if acquireSeq > dva.lastAssignedAcquirement || acquireSeq <= dva.lastProcessedAcquirement {
		return VolatileAcquirementInvalidSequenceError
	}
	fetchedAcquirement, ok := dva.PendingAcquirement[acquireSeq]
	if !ok {
		return VolatileAcquirementInfoFetchingError
	}

	// if the lock acquirement has already expired, you cannot refresh it
	if fetchedAcquirement.LastRefreshedTimestamp + fetchedAcquirement.Expire < timestamp {
		return VolatileAcquirementExpireError
	} else {
		// actual update process if lock acquirement is not expired
		fetchedAcquirement.LastRefreshedTimestamp = timestamp
		return nil
	}
}

// remove the acquirement expired, starting from sequence (dva.lastProcessedAcquirement + 1)
func (dva *DlockVolatileAcquirement) AbandonExpiredAcquirement(timestamp int64) error {

	// find all acquirement expired
	for i := dva.lastProcessedAcquirement + 1; i <= dva.lastAssignedAcquirement; i++{
		fetchedAcquirement, ok := dva.PendingAcquirement[i]
		if !ok {
			return VolatileAcquirementInfoFetchingError
		}
		if fetchedAcquirement.LastRefreshedTimestamp + fetchedAcquirement.Expire < timestamp {
			// delete expired acquirement
			delete(dva.PendingAcquirement, i)
			dva.lastProcessedAcquirement = i
		} else {
			// return when finding the first not expired acquirement
			return nil
		}
	}
	return nil
}

// insert a new acquirement
func (dva *DlockVolatileAcquirement) InsertNewAcquirement(command *storage.CommandDLock,
	timestamp int64, expire int64) (uint32, error) {

	// first refresh pending list
	err := dva.AbandonExpiredAcquirement(timestamp)
	if err != nil {
		return 0, err
	}

	// get dlock info from command
	dlockInfo, err := command.GetAsDLockInfo()
	if err != nil {
		return 0, err
	}
	// get current acquirer of dlock
	acquirer := dlockInfo.NewOwner

	// search the current pending list, figuring out whether the acquirer has already acquired this dlock
	for i := dva.lastProcessedAcquirement + 1; i <= dva.lastAssignedAcquirement; i++{
		fetchedAcquirement, ok := dva.PendingAcquirement[i]
		if !ok {
			return 0, VolatileAcquirementInfoFetchingError
		}
		// return the waiting acquirement in the pending list
		if fetchedAcquirement.Acquirer == acquirer {
			return i, nil
		}
	}

	// if we do not find an acquirement with the same acquirer, construct a pending acquirement
	dlockAcqInfo := NewDlockAcquirementInfo(acquirer, timestamp, expire, command)

	dva.lastAssignedAcquirement += 1
	dva.PendingAcquirement[dva.lastAssignedAcquirement] = dlockAcqInfo
	return dva.lastAssignedAcquirement, nil

}

// remember that this function will not increment dva.LastAppendedNonce
func (dva *DlockVolatileAcquirement) PopFirstValidAcquirement(timestamp int64) (*storage.CommandDLock, error) {

	// first refresh pending list
	err := dva.AbandonExpiredAcquirement(timestamp)
	if err != nil {
		return nil, err
	}

	if dva.lastAssignedAcquirement == dva.lastProcessedAcquirement {
		// if no valid pending acquirement is available, return nothing
		return nil, nil
	} else {
		// has one valid pending acquirement, return it and refresh lastProcessedAcquirement
		dva.lastProcessedAcquirement += 1
		fetchedAcquirement, ok := dva.PendingAcquirement[dva.lastProcessedAcquirement]
		if !ok {
			return nil, VolatileAcquirementInfoFetchingError
		}
		// refresh the timestamp in DLockCommand as ts in arg, ---often---> time.Now().NanoSeconds()
		fetchedDlockInfo, err := fetchedAcquirement.Command.GetAsDLockInfo()
		if err != nil {
			return nil, err
		}
		fetchedDlockInfo.Timestamp = timestamp
		fetchedDlockInfo.LockNonce = dva.LastAppendedNonce + 1
		err = fetchedAcquirement.Command.SetAsDLockInfo(fetchedDlockInfo)
		if err != nil {
			return nil, err
		}
		dva.lastProcessedAcquirement += 1
		return fetchedAcquirement.Command, nil
	}
}

// the information for an lock acquirement
type DlockAcquirementInfo struct {

	// the acquirement sender
	Acquirer string

	// the timestamp when acquirement sender last refreshed the acquirement
	// those who do not refresh the acquirement will finally expire
	LastRefreshedTimestamp int64

	// the expire for this dlockAcquirement, currently determined by server and cannot revise
	// used to judge whether this acquirement has expired
	Expire int64

	// the command recorded for this acquirement
	Command *storage.CommandDLock
}

func NewDlockAcquirementInfo(acquirer string, lastRefreshedTimestamp int64,
	expire int64, command *storage.CommandDLock) *DlockAcquirementInfo {
	return &DlockAcquirementInfo{
		Acquirer: acquirer,
		LastRefreshedTimestamp: lastRefreshedTimestamp,
		Expire: expire,
		Command: command,
	}
}

