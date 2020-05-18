package storage

import (
	"fmt"
	"testing"
	"time"
)

func TestKVStore(t *testing.T){
	
	// key value pending to be tested
	testKey := "TestKey"
	testValue := []byte("TestValue")

	// new KVStore command
	testKVCommand, err1 := NewCommandKVStore(testKey, testValue)
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a new KVStore command: %s", err1))
	}
	t.Log(fmt.Println("The original set KVStore: ", testKVCommand))
	
	// read kv info
	testKeyValue, err2 := testKVCommand.GetAsKVStore()
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when get KVStore from its Command: %s", err2))
	}
	t.Log(fmt.Println("The fetched KVStore: ", testKeyValue))
	
	// revise value
	testKeyValue.Value = []byte("ValueTest")
	testKeyValue.Key = "KeyTest"
	err3 := testKVCommand.SetAsKVStore(testKeyValue)
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when setting new KVStore to Command: %s", err3))
	}
	t.Log(fmt.Println("The newly set KVStoreCommand: ", testKVCommand))

	// test fast index
	fastIndex, err4 := testKVCommand.GetFastIndex()
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when getting fast index of KVStore: %s", err4))
	}
	t.Log(fmt.Printf("The Fast index: %s \n", fastIndex))

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
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a new DLock command: %s", err1))
	}
	t.Log(fmt.Println("The original DLockCommand: ", testDLockCommand))
	
	// read lock info
	testDLockInfo, err2 := testDLockCommand.GetAsDLockInfo()
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a new DLock command: %s", err2))
	}
	t.Log(fmt.Println("The fetched DLock Command Info: ", testDLockInfo))

	// revise value
	testDLockInfo.Timestamp = time.Now().UnixNano()
	err3 := testDLockCommand.SetAsDLockInfo(testDLockInfo)
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a new DLock command: %s", err3))
	}
	t.Log(fmt.Println("The newly set DLockCommand: ", testDLockCommand))

	// test fast index
	fastIndex, err4 := testDLockCommand.GetFastIndex()
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when getting fast index of DLock: %s", err4))
	}
	t.Log(fmt.Printf("The Fast index: %s \n", fastIndex))

}	