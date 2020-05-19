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
	QuerySpecificState(key string) ([]byte,error)
	UpdateStateFromLogEntry(logEntry *LogEntry) error
	UpdateStateFromLogMemory(logMemory *LogMemory, startIndex uint64, endIndex uint64) error

}

func (s *StateMapMemory) GetPrevIndex() uint64 {
	return s.prevIndex
}

func (s *StateMapMemory) GetPrevTerm() uint64 {
	return s.prevTerm
}

func (s *StateMapMemory) GetKeyword() string{
	return s.keyword
}

func (s *StateMapMemory) SetKeyword(keyword string) {
	s.keyword = keyword
	return
}
