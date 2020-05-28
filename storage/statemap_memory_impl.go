// an in memory statemap implementation
package storage

import (
	"encoding/json"
	"errors"
	"strings"
)

var InMemoryStateMapCreateError = errors.New("dlock_raft.statemap_memory: " +
	"StateMapKVStore should have a keyword as `KVStore`")
var InMemoryStateMapKeywordError = errors.New("dlock_raft.statemap_memory: " +
	"the LogEntry command Keyword is not valid")
var InMemoryStateMapIndexTermError = errors.New("dlock_raft.statemap_memory: " +
	"the LogEntry index/term is not valid")
var InMemoryStateMapContentError = errors.New("dlock_raft.statemap_memory: " +
	"the LogEntry command content is not valid")
var InMemoryStateMapFetchError = errors.New("dlock_raft.statemap_memory: " +
	"the no value for the specific key in statemap")
var InMemoryStateMapDLockOwnerMisMatchError = errors.New("dlock_raft.statemap_memory: " +
	"owner mismatch for dlock update")

type StateMapMemoryKVStore struct {
	StateMapMemory
}

// keyword should have prefix KVStore
func NewStateMapMemoryKVStore(keyword string) (*StateMapMemoryKVStore, error) {

	// note that by default, the stateMap will begin with empty current states
	stateMapMemory := new (StateMapMemoryKVStore)
	if !strings.HasPrefix(keyword, "KVStore"){
		return nil, InMemoryStateMapCreateError
	}
	stateMapMemory.keyword = keyword
	stateMapMemory.prevIndex = 0
	stateMapMemory.prevTerm = 0
	stateMapMemory.StateMap = make(map[string][]byte)
	return stateMapMemory, nil

}

func (sm *StateMapMemoryKVStore) UpdateStateFromLogEntry(entry *LogEntry) error {

	// do nothing if Entry object is nil, or keyword does not match
	if entry == nil {
		return nil
	}  else if (sm.prevIndex + 1) != entry.Entry.Index || sm.prevTerm > entry.Entry.Term {
		// note that LogEntry should be applied one by one in terms of index
		return InMemoryStateMapIndexTermError
	} else if !strings.HasPrefix(entry.GetFastIndex(), sm.keyword) {
		// does not fix fast index, then only process the index and term
		sm.prevIndex = entry.Entry.Index
		sm.prevTerm = entry.Entry.Term
		return InMemoryStateMapKeywordError
	}

	// update from LogEntry
	command := NewCommandFromRaw(entry.Entry.CommandName, entry.Entry.CommandContent)
	// check if the command a valid CommandKVStore
	commandKVStore, ok := command.(*CommandKVStore)
	if !ok {
		return InMemoryStateMapContentError
	}
	// get key value of commandKVStore
	kvStore, err := commandKVStore.GetAsKVStore()
	if err != nil {
		return err
	}

	// the update process
	// if value == nil, then it is a command for delete
	// else, it is a put command
	if kvStore.Value == nil {
		_, ok := sm.StateMap[kvStore.Key]
		// if there already exists a value for kvStore.Key, delete it
		if ok {
			delete(sm.StateMap, kvStore.Key)
		}
	} else {
		sm.StateMap[kvStore.Key] = kvStore.Value
	}
	sm.prevIndex = entry.Entry.Index
	sm.prevTerm = entry.Entry.Term

	return nil
}

func (sm *StateMapMemoryKVStore) QuerySpecificState(key string) (interface{}, error) {

	// should test whether a specific key-value is stored in statemap
	value, ok := sm.StateMap[key]
	if !ok {
		return nil, InMemoryStateMapFetchError
	}
	return value, nil

}

func (sm *StateMapMemoryKVStore) UpdateStateFromLogMemory(logMemory *LogMemory, start uint64, end uint64) error {

	// test whether the indexes are valid
	// update should be done after the previous index
	if start == 0 || end == 0 || start != sm.prevIndex + 1 ||
		start > end || end > logMemory.maximumIndex {
		return InMemoryStateMapIndexTermError
	}

	// update one by one
	sequence := start
	for true {
		logEntry, err := logMemory.FetchLogEntry(sequence)
		if err != nil {
			return err
		}
		err = sm.UpdateStateFromLogEntry(logEntry)
		if err != nil && err != InMemoryStateMapKeywordError {
			return err
		}
		sequence += 1
		if sequence > end {break}
	}
	return nil

}


type StateMapMemoryDLock struct {
	StateMapMemory
}

// the dlock state currently
type DlockState struct {
	// current owner
	Owner string `json:"owner"`
	// LockId should be unique
	LockId uint32 `json:"lock_id"`
	// LockName should also be unique, will be used as fastIndex
	LockName string `json:"lock_name"`
	// last modified timestamp
	Timestamp int64 `json:"timestamp"`
}

// keyword should have prefix DLock
func NewStateMapMemoryDLock(keyword string) (*StateMapMemoryDLock, error) {

	// note that by default, the stateMap will begin with empty current states
	stateMapMemory := new (StateMapMemoryDLock)
	if !strings.HasPrefix(keyword, "DLock"){
		return nil, InMemoryStateMapCreateError
	}
	stateMapMemory.keyword = keyword
	stateMapMemory.prevIndex = 0
	stateMapMemory.prevTerm = 0
	stateMapMemory.StateMap = make(map[string][]byte)
	return stateMapMemory, nil

}

func (sm *StateMapMemoryDLock) UpdateStateFromLogEntry(entry *LogEntry) error {

	// do nothing if Entry object is nil, or keyword does not match
	if entry == nil {
		return nil
	}  else if (sm.prevIndex + 1) != entry.Entry.Index || sm.prevTerm > entry.Entry.Term {
		// note that LogEntry should be applied one by one in terms of index
		return InMemoryStateMapIndexTermError
	} else if !strings.HasPrefix(entry.GetFastIndex(), sm.keyword) {
		// does not fix fast index, then only process the index and term
		sm.prevIndex = entry.Entry.Index
		sm.prevTerm = entry.Entry.Term
		return InMemoryStateMapKeywordError
	}

	// update from LogEntry
	command := NewCommandFromRaw(entry.Entry.CommandName, entry.Entry.CommandContent)
	// check if the command a valid CommandDLock
	commandDLock, ok := command.(*CommandDLock)
	if !ok {
		return InMemoryStateMapContentError
	}
	// get key value of commandDLock
	dlockInfo, err := commandDLock.GetAsDLockInfo()
	if err != nil {
		return err
	}

	// query the current state
	currentDlockState, err2 := sm.QuerySpecificState(dlockInfo.LockName)
	if err2 != InMemoryStateMapFetchError && err2 != nil {
		return err2
	}
	// if dlock exists but current owner in LogEntry is not equal to the old owner, update Dlock should fail
	if err2 == nil {
		lockStatePrevious, ok := currentDlockState.(*DlockState)
		if !ok {
			return InMemoryStateMapContentError
		}
		if dlockInfo.OrigOwner != lockStatePrevious.Owner && dlockInfo.OrigOwner != "Unknown"{
			return InMemoryStateMapDLockOwnerMisMatchError
		}
	}

	// construct a new Dlock state
	// dlockState object object pending to be marshalled
	lockState := new(DlockState)
	lockState.LockName = dlockInfo.LockName
	lockState.LockId = dlockInfo.LockId
	lockState.Owner = dlockInfo.NewOwner
	lockState.Timestamp = dlockInfo.Timestamp

	// marshal the dlockState object
	encodedLockState, err2 := json.Marshal(lockState)
	if err2 != nil {
		return err2
	}

	// the update process
	sm.StateMap[dlockInfo.LockName] = encodedLockState
	sm.prevIndex = entry.Entry.Index
	sm.prevTerm = entry.Entry.Term

	return nil
}

func (sm *StateMapMemoryDLock) QuerySpecificState(key string) (interface{}, error) {

	// should test whether a specific key-value is stored in statemap
	encodedDlockState, ok := sm.StateMap[key]
	if !ok {
		return nil, InMemoryStateMapFetchError
	}

	// unmarshal the []byte array in statemap
	dlockState := new(DlockState)
	err := json.Unmarshal(encodedDlockState, dlockState)
	if err != nil {
		return nil, err
	}

	return dlockState, nil
}

func (sm *StateMapMemoryDLock) UpdateStateFromLogMemory(logMemory *LogMemory, start uint64, end uint64) error {

	// test whether the indexes are valid
	// update should be done after the previous index
	if start == 0 || end == 0 || start != sm.prevIndex + 1 ||
		start > end || end > logMemory.maximumIndex {
		return InMemoryStateMapIndexTermError
	}

	// update one by one
	sequence := start
	for true {
		logEntry, err := logMemory.FetchLogEntry(sequence)
		if err != nil {
			return err
		}
		err = sm.UpdateStateFromLogEntry(logEntry)
		if err != nil && err != InMemoryStateMapKeywordError {
			return err
		}
		sequence += 1
		if sequence > end {break}
	}
	return nil
}

