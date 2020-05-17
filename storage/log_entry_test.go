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

	// Log entry info
	testTerm := uint64(4000)
	testIndex := uint64(9225)
	testCommandKVStore, err1 := NewCommandKVStore("key", []byte("value"))
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a KVStore Command: %s\n", err1))
	}

	// create the new Log Entry entity
	testLogEntry := NewLogEntry(testTerm, testIndex, testCommandKVStore)
	// encode LogEntry with protobuf
	encodedTestLogEntry, err2 := testLogEntry.EncodeLogEntry()
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when encoding LogEntry with protobuf: %s\n", err2))
	}

	// decode LogEntry from protobuf
	recoverLogEntry := new(LogEntry)
	err3 := recoverLogEntry.DecodeLogEntry(encodedTestLogEntry)
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when decoding LogEntry with protobuf: %s\n", err3))
	}
	t.Log(fmt.Printf("The recovered LogEntry has attributes: %d, %d, %s \n",
		recoverLogEntry.entry.GetTerm(), recoverLogEntry.entry.GetIndex(),
		recoverLogEntry.entry.GetCommandName()))

}

func TestLogEntryWriteRead(t *testing.T) {

	// test term and index
	testTerm := uint64(4000)
	testIndex := uint64(9225)

	// Test Log entry1: KVStore
	testCommandKVStore, err1 := NewCommandKVStore("key", []byte("value"))
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a KVStore Command: %s\n", err1))
	}
	// create the new Log Entry entity
	testLogEntryKVStore := NewLogEntry(testTerm, testIndex, testCommandKVStore)

	// Test Log entry: DLock
	testLockId := uint32(9226)
	testLockName := "Lock_Calligrapher"
	testOrigOwner := "Golang"
	testNewOwner := "Java"
	testTimestamp := time.Now().UnixNano()
	testCommandDLock, err2 := NewCommandDLock(testLockId, testLockName,
		testOrigOwner, testNewOwner, testTimestamp)
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a DLock command: %s\n", err2))
	}
	// create the new Log Entry entity
	testLogEntryDLock := NewLogEntry(testTerm+1000, testIndex+1000, testCommandDLock)


	// create a temporary io writer / reader
	testTempFile, err3 := ioutil.TempFile("", "testFile")
	if err3 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a temp file: %s\n", err3))
	}
	defer os.Remove(testTempFile.Name())

	// write LogEntry to temporary file
	writtenBytes1, err4 := testLogEntryKVStore.LogStore(testTempFile)
	if err4 != nil {
		t.Error(fmt.Sprintf("Error happens when writting LogEntry KVStore: %s\n", err4))
	}
	t.Log(fmt.Printf("Written %d bytes for LogEntry KVStore\n", writtenBytes1))
	writtenBytes2, err5 := testLogEntryDLock.LogStore(testTempFile)
	if err5 != nil {
		t.Error(fmt.Sprintf("Error happens when writting LogEntry KVStore: %s\n", err5))
	}
	t.Log(fmt.Printf("Written %d bytes for LogEntry KVStore\n", writtenBytes2))


	// switch the read head to the head of the file
    _, err6 := testTempFile.Seek(0,0)
	if err6 != nil {
		t.Error(fmt.Sprintf("Error happens when set read head tempFile: %s", err6))
	}

	// read LogEntry KVStore from file
	recoverLogEntry1 := new(LogEntry)
	readBytes1, err7 := recoverLogEntry1.LogReload(testTempFile)
	if err6 != nil {
		t.Error(fmt.Sprintf("Error happens when reading LogEntry from tempFile: %s", err7))
	}
	t.Log(fmt.Printf("Read %d bytes for LogEntry KVStore\n", readBytes1))
	t.Log(fmt.Printf("The recovered LogEntry KVStore has attributes: %d, %d, %s \n",
		recoverLogEntry1.entry.GetTerm(), recoverLogEntry1.entry.GetIndex(),
		recoverLogEntry1.entry.GetCommandName()))

	// read LogEntry DLock from file
	recoverLogEntry2 := new(LogEntry)
	readBytes2, err8 := recoverLogEntry2.LogReload(testTempFile)
	if err7 != nil {
		t.Error(fmt.Sprintf("Error happens when decoding LogEntry from tempFile: %s", err8))
	}
	t.Log(fmt.Printf("Read %d bytes for LogEntry DLock\n", readBytes2))
	t.Log(fmt.Printf("The recovered LogEntry DLock has attributes: %d, %d, %s \n",
		recoverLogEntry2.entry.GetTerm(), recoverLogEntry2.entry.GetIndex(),
		recoverLogEntry2.entry.GetCommandName()))

	// the next read should return io.EOF, since there are no LogEntries any more
	recoverLogEntry3 := new(LogEntry)
	readBytes3, err9 := recoverLogEntry3.LogReload(testTempFile)
	t.Log(fmt.Printf("Read %d bytes\n", readBytes3))
	t.Log(fmt.Printf("Error equals to io.EOF (%t)\n", err9 == io.EOF))
}