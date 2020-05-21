// an in memory log implementation
package storage

import (
	"errors"
	"io"
)

var InMemoryLogInsertError = errors.New("dlock_raft.log_memory: insert Entry in in-memory log fails")
var InMemoryLogStoreError = errors.New("dlock_raft.log_memory: store in-memory log fails")
var InMemoryLogEntryNotExistError = errors.New("dlock_raft.log_memory: the required log Entry does not exist")

type LogMemory struct {
	// the hashmap for LogEntries in memory
	// the key for hashmap is the Entry index
	logInMemory map[uint64]*LogEntry

	// the maximum index which should not exceed
	maximumIndex uint64
}

func NewLogMemory() *LogMemory{

	logMemory := new(LogMemory)
	logMemory.logInMemory = make(map[uint64]*LogEntry)
	// the init maximum index is set to 0
	logMemory.maximumIndex = 0
	return logMemory

}

func (lm *LogMemory) MaximumIndex() uint64 {
	return lm.maximumIndex
}

func (lm *LogMemory) InsertLogEntry (entry *LogEntry) error {

	// Entry should not be nil
	if entry == nil || entry.Entry == nil{
		return InMemoryLogInsertError
	}

	// get the index for LogEntry as key for hashmap
	index := entry.Entry.GetIndex()
	// insert the LogEntry
	lm.logInMemory[index] = entry

	// change maximum index if necessary
	if entry.Entry.Index > lm.maximumIndex{
		lm.maximumIndex = entry.Entry.Index
	}

	// return nil if no error
	return nil

}

func (lm *LogMemory) FetchLogEntry (index uint64) (*LogEntry, error) {

	// if index > maximum index, directly return error
	if index > lm.maximumIndex {
		return nil, InMemoryLogEntryNotExistError
	}

	// get the required LogEntry
	logEntryRequired, ok := lm.logInMemory[index]
	if !ok {
		return nil, InMemoryLogEntryNotExistError
	}

	return logEntryRequired, nil
}

// insert a list if LogEntry in the current LogMemory
func (lm *LogMemory) InsertValidEntryList(entryList []*LogEntry) error {

	// insert LogEntry one by one
	for _, entry := range entryList {
		err := lm.InsertLogEntry(entry)
		if err != nil {
			return err
		}
	}
	return nil
}

// store the log memory to writer
// start and end are the front-rear indexes for store
// return the written bytes and potential errors
// Warning: the return length does not count the length characters and "\n" !!!!!!
func (lm *LogMemory) StoreLogMemory(start uint64, end uint64, writer io.Writer) (int, error){

	// test whether the indexes are valid
	if start == 0 || end == 0 || start > lm.maximumIndex || end > lm.maximumIndex || start > end {
		return 0, InMemoryLogStoreError
	}

	// store every object
	var entry *LogEntry
	var err1, err2 error
	var currentBytesWritten int
	totalBytesWritten := 0
	// the main loop for writing entries
	for i := start; i <= end; i++ {
		// fetch the log Entry
		entry, err1 = lm.FetchLogEntry(i)
		if err1 != nil {
			return 0, err1
		}
		// write every byte of LogEntry
		currentBytesWritten, err2 = entry.LogStore(writer)
		if err2 != nil {
			return totalBytesWritten, err2
		}
		// accumulate the bytes length
		totalBytesWritten += currentBytesWritten
	}

	return totalBytesWritten, nil
}

// recover the entire LogMemory from scratch
// used for sudden crashes of server, and should recover the entire log from scratch
func (lm *LogMemory) LogReload(reader io.Reader) (int, error) {

	// variables
	var currentReadBytes int
	var err error
	// read LogEntry DLock from file
	recoverLogEntry := new(LogEntry)
	totalReadBytes := 0

	for true {
		// try to read an Entry
		currentReadBytes, err = recoverLogEntry.LogReload(reader)

		// if EOF, then the recovering process is over
		if err == io.EOF {
			return totalReadBytes, nil
		} else if err != nil {
			return totalReadBytes, err
		}

		// fill in the Entry in hashmap
		lm.logInMemory[recoverLogEntry.Entry.Index] = recoverLogEntry
		// refine the maximum index
		if recoverLogEntry.Entry.Index > lm.maximumIndex {
			lm.maximumIndex = recoverLogEntry.Entry.Index
		}
		// accumulate the total read bytes
		totalReadBytes += currentReadBytes
	}

	return totalReadBytes, nil
}
