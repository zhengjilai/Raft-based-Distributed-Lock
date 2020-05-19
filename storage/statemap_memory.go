// an in memory statemap interface
package storage

type StateMapMemory struct {

	// the key word for this statemap, e.g. DLock, KVStore, DLock:LockName
	// often identical to command Name
	// this state map will only care about the Commands with this specific keyword
	keyword string

	// the actual state map, string -> []byte
	StateMap map[string][]byte

	// previous index and previous term
	// used to specify the actual state
	prevIndex uint64
	prevTerm uint64
}

// interface for in-memory statemap
type StateMapMemoryOperator interface{

	GetPrevIndex() uint64
	GetPrevTerm() uint64
	GetKeyword() string
	SetKeyword(keyword string)
	QuerySpecificState(key string) (interface{}, error)
	UpdateStateFromLogEntry(logEntry *LogEntry) error
	UpdateStateFromLogMemory(logMemory *LogMemory, start uint64, end uint64) error

}

func (sm *StateMapMemory) GetPrevIndex() uint64 {
	return sm.prevIndex
}

func (sm *StateMapMemory) GetPrevTerm() uint64 {
	return sm.prevTerm
}

func (sm *StateMapMemory) GetKeyword() string{
	return sm.keyword
}

func (sm *StateMapMemory) SetKeyword(keyword string) {
	sm.keyword = keyword
	return
}
