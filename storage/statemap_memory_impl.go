// an in memory statemap implementation
package storage

import (
	"errors"
	"strings"
)

var InMemoryStateMapKVStoreCreateError = errors.New("dlock_raft.statemap_memory: " +
	"StateMapKVStore should have a keyword as `KVStore`")


type StateMapMemoryKVStore struct {
	StateMapMemory
}

// keyword should have prefix KVStore
func NewStateMapMemoryKVStore(keyword string) (*StateMapMemory, error) {

	// note that by default, the statemap will begin with empty current states
	stateMapMemory := new (StateMapMemory)
	if !strings.HasPrefix(keyword, "KVStore"){
		return nil, InMemoryStateMapKVStoreCreateError
	}
	stateMapMemory.keyword = keyword
	stateMapMemory.prevIndex = 0
	stateMapMemory.prevTerm = 0
	stateMapMemory.StateMap = make(map[string][]byte)
	return stateMapMemory, nil

}

func (*StateMapMemory) UpdateStateFromLogEntry(entry *LogEntry) (error) {

	// do nothing if entry object is nil
	if entry == nil {
		return nil
	}
	// update from LogEntry
	// fastIndex := entry.fastIndex


}
