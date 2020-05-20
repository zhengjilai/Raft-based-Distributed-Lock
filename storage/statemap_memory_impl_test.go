package storage

import (
	"fmt"
	"testing"
	"time"
)

func CreateTestLogMemory(t *testing.T) *LogMemory {

	// the new LogMemory, we will update the statemap based on LogMemory
	testLogMemory := NewLogMemory()

	// keys and values for KVStore
	testKeys := []string{"key1", "key2", "key1", "key3", "key2"}
	testValues := []string{"value1", "value2", "value1-u", "value3", "value2-u"}
	indexesKV := []uint64{1,2,3,4,5}
	termsKV := []uint64{4,5,6,6,6}

	// insert the KVStore commands into LogMemory
	for i := 0; i < len(testKeys) ; i++ {
		command, err1 := NewCommandKVStore(testKeys[i], []byte(testValues[i]))
		if err1 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a KVStore Command: %s\n", err1))
		}
		logEntry, err2 := NewLogEntry(termsKV[i], indexesKV[i], command)
		if err2 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a LogEntry: %s\n", err2))
		}
		err3 := testLogMemory.InsertLogEntry(logEntry)
		if err3 != nil {
			t.Error(fmt.Sprintf("Error happens when inserting a LogEntry: %s\n", err3))
		}
	}

	// information for dlock
	lockId := []uint32{1, 2, 1, 2, 1}
	// LockName should be unique, will be used as fastIndex
	lockName := []string{"DLock1", "DLock2", "DLock1", "DLock2", "DLock1"}
	origOwner := []string{"", "", "peer1", "Unknown", "peer2"}
	newOwner := []string{"peer1","peer1","peer2","peer4","peer9"}
	now := time.Now().UnixNano()
	timestamp := []int64 {now, now + 1, now + 2, now + 3, now + 4}
	indexesDLock := []uint64{6,7,8,9,10}
	termsDLock := []uint64{6,6,6,7,7}

	// insert the DLock commands into LogMemory
	for i := 0; i < len(lockId); i++ {
		command, err1 := NewCommandDLock(lockId[i], lockName[i], origOwner[i], newOwner[i], timestamp[i])
		if err1 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a DLock Command: %s\n", err1))
		}
		logEntry, err2 := NewLogEntry(termsDLock[i], indexesDLock[i], command)
		if err2 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a LogEntry: %s\n", err2))
		}
		err3 := testLogMemory.InsertLogEntry(logEntry)
		if err3 != nil {
			t.Error(fmt.Sprintf("Error happens when inserting a LogEntry: %s\n", err3))
		}
	}

	return testLogMemory

}

func TestStateMapKVStoreOperations(t *testing.T) {

	// test log memory generator
	testLogMemory := CreateTestLogMemory(t)

	// a new StateMapMemoryKVStore
	stateMapMemoryKVStore, err1 := NewStateMapMemoryKVStore("KVStore")
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a StateMapMemoryKVStore: %s\n", err1))
	}

	err2 := stateMapMemoryKVStore.UpdateStateFromLogMemory(testLogMemory, 1, 10)
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when updating state from LogMemory: %s\n", err2))
	}

	t.Log(fmt.Println("The current state: \n", stateMapMemoryKVStore))
}

func TestStateMapDLockOperations(t *testing.T) {

	// test log memory generator
	testLogMemory := CreateTestLogMemory(t)

	// a new StateMapMemoryKVStore
	stateMapMemoryDLock, err1 := NewStateMapMemoryDLock("DLock:DLock1")
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a StateMapMemoryDLock: %s\n", err1))
	}

	err2 := stateMapMemoryDLock.UpdateStateFromLogMemory(testLogMemory, 1, 10)
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when updating state from LogMemory: %s\n", err2))
	}

	// query a specific DLock
	testDLock, err3 := stateMapMemoryDLock.QuerySpecificState("DLock2")
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when query state from LogMemory: %s\n", err3))
	}
	testDLockFormat, ok := testDLock.(*DlockState)
	if !ok {
		t.Error(fmt.Sprintf("Error happens since the queried state from LogMemory has wrong format.\n"))
	} else {
		t.Log(fmt.Printf("The current state for %s:\n", testDLockFormat.LockName))
		t.Log(fmt.Println(testDLockFormat))
	}
}