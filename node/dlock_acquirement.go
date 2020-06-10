package node

import (
	"errors"
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
	
	// map: acquire seq -> actual Dlock acquirements
	PendingAcquirement map[uint32]*DlockAcquirementInfo
	
}

func NewDlockVolatileAcquirement(lockName string) *DlockVolatileAcquirement {
	return &DlockVolatileAcquirement{
		LockName: lockName,
		lastAssignedAcquirement: 0,
		lastProcessedAcquirement: 0,
		PendingAcquirement: make(map[uint32]*DlockAcquirementInfo)}
}

// update an requirement with current timestamp, in case it expires
// time stamp format ms: obtained by time.Now().UnixNano() / 1000000
func (dva *DlockVolatileAcquirement) RefreshAcquirement(acquireSeq uint32, timestamp uint64) error {

	// if the sequence is not valid
	if acquireSeq > dva.lastAssignedAcquirement || acquireSeq <= dva.lastProcessedAcquirement {
		return VolatileAcquirementInvalidSequenceError
	}
	fetchedAcquirement, ok := dva.PendingAcquirement[acquireSeq]
	if !ok {
		return VolatileAcquirementInfoFetchingError
	}

	// if the lock acquirement has already expired
	if fetchedAcquirement.LastRefreshedTimestamp + fetchedAcquirement.Expire < timestamp {
		// update processed sequence record if possible
		if acquireSeq == dva.lastProcessedAcquirement + 1{
			dva.lastProcessedAcquirement += 1
		}
		return VolatileAcquirementExpireError
	} else {
		// actual update process
		fetchedAcquirement.LastRefreshedTimestamp = timestamp
		return nil
	}
}

// remove the acquirement expired, starting from sequence (dva.lastProcessedAcquirement + 1)
func (dva *DlockVolatileAcquirement) AbandonAllExpired() error {
	return nil
}


// insert a new acquirement
func (dva *DlockVolatileAcquirement) InsertNewAcquirement(timestamp uint64,
	expire uint64, acquirer uint64) {

}

// the information for an lock acquirement
type DlockAcquirementInfo struct {

	// the acquirement sender
	Acquirer string

	// the timestamp when acquirement sender last refreshed the acquirement
	// those who do not refresh the acquirement will finally expire
	LastRefreshedTimestamp uint64

	// the expire for this dlockAcquirement, currently determined by server and cannot revise
	// used to judge whether this acquirement has expired
	Expire uint64

}

func NewDlockAcquirementInfo(acquirer string, lastRefreshedTimestamp uint64,
	expire uint64) *DlockAcquirementInfo {
	return &DlockAcquirementInfo{
		Acquirer: acquirer,
		LastRefreshedTimestamp: lastRefreshedTimestamp,
		Expire: expire,
	}
}

