package storage

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
)

func TestLogEntryMemoryBasicOperations(t *testing.T) {

	// the new LogMemory
	testLogMemory := NewLogMemory()
	// keys
	testKeys := []string{"key1", "key2", "key3", "key4", "key5"}
	testValues := []string{"value1", "value2", "value3", "value4", "value5"}

	// insert the KVStore commands into LogMemory
	for i := 0; i < len(testKeys); i++ {
		command, err1 := NewCommandKVStore(testKeys[i], []byte(testValues[i]))
		if err1 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a KVStore Command: %s\n", err1))
		}
		logEntry, err2 := NewLogEntry(uint64(1001+i), uint64(i+1), command)
		if err2 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a LogEntry: %s\n", err2))
		}
		err3 := testLogMemory.InsertLogEntry(logEntry)
		if err3 != nil {
			t.Error(fmt.Sprintf("Error happens when inserting a LogEntry: %s\n", err3))
		}
	}

	// test a valid entry
	recoverLogEntry, err4 := testLogMemory.FetchLogEntry(3)
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when getting a LogEntry: %s\n", err4))
	}
	t.Log(fmt.Printf("The read valid LogEntry has attributes: %d, %d, %s \n",
		recoverLogEntry.entry.GetTerm(), recoverLogEntry.entry.GetIndex(),
		recoverLogEntry.entry.GetCommandName()))

	// test an invalid entry
	_, err5 := testLogMemory.FetchLogEntry(9225)
	if err5 != nil {
		t.Log(fmt.Sprintf("Wanted error happens when getting an invalid LogEntry: %s\n", err5))
	} else {
		t.Error(fmt.Sprintf("Unwanted error happens when getting an invalid LogEntry: %s\n", err5))
	}

}

func TestLogEntryMemoryStoreRecover(t *testing.T) {

	// the new LogMemory
	testLogMemory := NewLogMemory()
	// keys
	testKeys := []string{"key1", "key2", "key3", "key4", "key5"}
	testValues := []string{"value1", "value2", "value3", "value4", "value5"}

	// insert the KVStore commands into LogMemory
	for i := 0; i < len(testKeys); i++ {
		command, err1 := NewCommandKVStore(testKeys[i], []byte(testValues[i]))
		if err1 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a KVStore Command: %s\n", err1))
		}
		logEntry, err2 := NewLogEntry(uint64(1001+i), uint64(1+i), command)
		if err2 != nil {
			t.Error(fmt.Sprintf("Error happens when creating a LogEntry: %s\n", err2))
		}
		err2 = testLogMemory.InsertLogEntry(logEntry)
		if err2 != nil {
			t.Error(fmt.Sprintf("Error happens when inserting a LogEntry: %s\n", err2))
		}
	}

	// create a temporary io writer / reader
	testTempFile, err3 := ioutil.TempFile("", "testFile")
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a temp file: %s\n", err3))
	}
	defer os.Remove(testTempFile.Name())

	// store the LogEntry in tempfile
	// first stage: write 1-3
	totalWrittenBytes, err4 := testLogMemory.StoreLogMemory(1,3, testTempFile)
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when storing the log memory: %s\n", err4))
	}
	t.Log(fmt.Sprintf("Total written bytes for index from %d to %d: %d\n", 1, 3, totalWrittenBytes))
	// second stage: write 4-5
	totalWrittenBytes, err4 = testLogMemory.StoreLogMemory(4,5, testTempFile)
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when storing the log memory: %s\n", err4))
	}
	t.Log(fmt.Sprintf("Total written bytes for index from %d to %d: %d\n", 4, 5, totalWrittenBytes))

	// recover from the log file
	// switch the read head to the head of the file
	_, err5 := testTempFile.Seek(0,0)
	if err5 != nil {
		t.Error(fmt.Sprintf("Error happens when set read head tempFile: %s", err5))
	}

	// actual recover process
	recoverLogMemory := NewLogMemory()
	totalReadBytes, err6 := recoverLogMemory.LogReload(testTempFile)
	if err6 != nil {
		t.Error(fmt.Sprintf("Error happens when storing the log memory: %s\n", err6))
	}
	// Assert (totalReadBytes == totalReadBytes + (8+1) * len(testKeys))
	t.Log(fmt.Sprintf("Total read bytes for index from 1 to %d: %d\n", recoverLogMemory.maximumIndex, totalReadBytes))

	// test some entries
	// test a valid entry
	recoverLogEntry, err7 := testLogMemory.FetchLogEntry(3)
	if err7 != nil {
		t.Error(fmt.Sprintf("Error happens when getting a LogEntry: %s\n", err7))
	}
	t.Log(fmt.Printf("The read valid LogEntry has attributes: %d, %d, %s \n",
		recoverLogEntry.entry.GetTerm(), recoverLogEntry.entry.GetIndex(),
		recoverLogEntry.entry.GetCommandName()))

	// test an invalid entry
	recoverLogEntry, err8 := testLogMemory.FetchLogEntry(9225)
	if err8 != nil {
		t.Log(fmt.Sprintf("Wanted error happens when getting an invalid LogEntry: %s\n", err8))
	} else {
		t.Error(fmt.Sprintf("Unwanted error happens when getting an invalid LogEntry: %s\n", err8))
	}

}