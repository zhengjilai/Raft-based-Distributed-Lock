package storage

import (
	"testing"
	"fmt"
	"time"
)

func TestKVStore(t *testing.T){
	
	// key value pending to be tested
	testKey := "TestKey"
	testValue := []byte("TestValue")

	// new KVStore command
	testKVComannd, err1 := NewCommandKVStore(testKey, testValue)
	
	// read kv info
	testKeyValue, err2 := testKVComannd.GetAsKVStore()
	fmt.Println(testKeyValue)
	
	// revise value
	testKeyValue.Value = []byte("ValueTest")
	err3 := testKVComannd.SetAsKVStore(testKeyValue)
	fmt.Println(testKeyValue)

	fmt.Println(err1, err2, err3)
}

func TestDLock(t *testing.T){
	
	// Dlock info pending to be tested
	testLockId := uint32(9225)
	testLockName := "Lock_Calligrapher"
	testOrigOwner := "Golang"
	testNewOwner := "Java"
	testTimestamp := time.Now().UnixNano()

	// new command
	testDLockCommand, err1 := NewCommandDLock(testLockId, testLockName,
		 testOrigOwner, testNewOwner, testTimestamp)
	
	// read lock info
	testDLockInfo, err2 := testDLockCommand.GetAsDLockInfo()
	fmt.Println(testDLockInfo)

	// revise value
	testDLockInfo.Timestamp = time.Now().UnixNano()
	err3 := testDLockCommand.SetAsDLockInfo(testDLockInfo)
	fmt.Println(testDLockInfo)

	fmt.Println(err1, err2, err3)
}	