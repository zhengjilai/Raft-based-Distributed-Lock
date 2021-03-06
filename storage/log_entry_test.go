package storage

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestLogEntryEncodeDecode(t *testing.T) {

	// Log Entry info
	testTerm := uint64(4000)
	testIndex := uint64(9225)
	testCommandKVStore, err1 := NewCommandKVStore("key", []byte("value"))
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a KVStore Command: %s\n", err1))
	}

	// create the new Log Entry entity
	testLogEntry, err2 := NewLogEntry(testTerm, testIndex, testCommandKVStore)
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a LogEntry: %s\n", err2))
	}

	// encode LogEntry with protobuf
	encodedTestLogEntry, err3 := testLogEntry.EncodeLogEntry()
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when encoding LogEntry with protobuf: %s\n", err3))
	}

	// decode LogEntry from protobuf
	recoverLogEntry := new(LogEntry)
	err4 := recoverLogEntry.DecodeLogEntry(encodedTestLogEntry)
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when decoding LogEntry with protobuf: %s\n", err3))
	}
	t.Log(fmt.Printf("The recovered LogEntry has attributes: %d, %d, %s, %s \n",
		recoverLogEntry.Entry.GetTerm(), recoverLogEntry.Entry.GetIndex(),
		recoverLogEntry.Entry.GetCommandName(), recoverLogEntry.GetFastIndex()))

}

func TestLogEntryWriteRead(t *testing.T) {

	// test term and index
	testTerm := uint64(4000)
	testIndex := uint64(9225)

	// Test Log Entry 1: KVStore
	testCommandKVStore, err1 := NewCommandKVStore("key", []byte("value"))
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a KVStore Command: %s\n", err1))
	}
	// create the new Log Entry entity
	testLogEntryKVStore, err2 := NewLogEntry(testTerm, testIndex, testCommandKVStore)
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a LogEntry: %s\n", err2))
	}


	// Test Log Entry 2: DLock
	testLockNonce := uint32(9226)
	testLockName := "Lock_Calligrapher"
	testNewOwner := "Java"
	testTimestamp := time.Now().UnixNano()
	testExpire := time.Duration(1).Nanoseconds()
	testCommandDLock, err3 := NewCommandDLock(testLockNonce, testLockName, testNewOwner,
		testTimestamp, testExpire)
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a DLock command: %s\n", err2))
	}
	// create the new Log Entry entity
	testLogEntryDLock, err4 := NewLogEntry(testTerm+1000, testIndex+1000, testCommandDLock)
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a LogEntry: %s\n", err4))
	}


	// create a temporary io writer / reader
	testTempFile, err5 := ioutil.TempFile("", "testFile")
	if err5 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a temp file: %s\n", err5))
	}
	defer os.Remove(testTempFile.Name())


	// write LogEntry to temporary file
	writtenBytes1, err6 := testLogEntryKVStore.LogStore(testTempFile)
	if err6 != nil {
		t.Error(fmt.Sprintf("Error happens when writting LogEntry KVStore: %s\n", err6))
	}
	t.Log(fmt.Printf("Written %d bytes for LogEntry KVStore\n", writtenBytes1))
	writtenBytes2, err7 := testLogEntryDLock.LogStore(testTempFile)
	if err7 != nil {
		t.Error(fmt.Sprintf("Error happens when writting LogEntry KVStore: %s\n", err7))
	}
	t.Log(fmt.Printf("Written %d bytes for LogEntry KVStore\n", writtenBytes2))


	// switch the read head to the head of the file
    _, err8 := testTempFile.Seek(0,0)
	if err8 != nil {
		t.Error(fmt.Sprintf("Error happens when set read head tempFile: %s", err8))
	}

	// read LogEntry KVStore from file
	recoverLogEntry1 := new(LogEntry)
	readBytes1, err9 := recoverLogEntry1.LogReload(testTempFile)
	if err6 != nil {
		t.Error(fmt.Sprintf("Error happens when reading LogEntry from tempFile: %s", err9))
	}
	t.Log(fmt.Printf("Read %d bytes for LogEntry KVStore\n", readBytes1))
	t.Log(fmt.Printf("The recovered LogEntry KVStore has attributes: %d, %d, %s \n",
		recoverLogEntry1.Entry.GetTerm(), recoverLogEntry1.Entry.GetIndex(),
		recoverLogEntry1.Entry.GetCommandName()))

	// read LogEntry DLock from file
	recoverLogEntry2 := new(LogEntry)
	readBytes2, err10 := recoverLogEntry2.LogReload(testTempFile)
	if err7 != nil {
		t.Error(fmt.Sprintf("Error happens when decoding LogEntry from tempFile: %s", err10))
	}
	t.Log(fmt.Printf("Read %d bytes for LogEntry DLock\n", readBytes2))
	t.Log(fmt.Printf("The recovered LogEntry DLock has attributes: %d, %d, %s, %s \n",
		recoverLogEntry2.Entry.GetTerm(), recoverLogEntry2.Entry.GetIndex(),
		recoverLogEntry2.Entry.GetCommandName(), recoverLogEntry2.GetFastIndex()))

	// the next read should return io.EOF, since there are no LogEntries any more
	recoverLogEntry3 := new(LogEntry)
	readBytes3, err11 := recoverLogEntry3.LogReload(testTempFile)
	t.Log(fmt.Printf("Read %d bytes\n", readBytes3))
	t.Log(fmt.Printf("Error equals to io.EOF (%t)\n", err11 == io.EOF))
}