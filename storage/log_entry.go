package storage

import (
	"errors"
	"fmt"
	"github.com/dlock_raft/protobuf"
	"google.golang.org/protobuf/proto"
	"io"
)

var LogEntryContentDecodeError = errors.New("dlock_raft.log_entry: " +
	"command content decode fails, the read []byte is invalid")
var LogEntryFastIndexError = errors.New("dlock_raft.log_entry: " +
	"command content decode fails, the read []byte is invalid")

// A log Entry type for storing, encoding and decoding entries
type LogEntry struct {
	Entry *protobuf.Entry

	// fast index is used for quickly recover state for a single key/dlock
	// it often contains the key of KVStore, or the LockName of DLock
	// it can also be extended to other usages
	fastIndex string
}

func NewLogEntry(term uint64, index uint64, command CommandOperator) (*LogEntry, error) {

	// elements are stored in protobuf.Entry
	pbEntry := new(protobuf.Entry)
	pbEntry.Term = term
	pbEntry.Index = index
	pbEntry.CommandName = command.GetCommandName()
	pbEntry.CommandContent = command.GetCommandContent()

	// the feedback LogEntry entity
	logEntry := new(LogEntry)
	logEntry.Entry = pbEntry

	// set fast index for LogEntry
	fastIndex, err := command.GetFastIndex()
	if err != nil {
		return nil, err
	}
	logEntry.fastIndex = fastIndex

    return logEntry, nil
}

func NewLogEntryList (pbEntry *protobuf.Entry) (*LogEntry,error) {
	command := NewCommandFromRaw(pbEntry.CommandName, pbEntry.CommandContent)
	fastIndex, err := command.GetFastIndex()
	if err != nil {
		return nil, err
	}
	// the feedback LogEntry entity
	logEntry := new(LogEntry)
	logEntry.Entry = pbEntry
	logEntry.fastIndex = fastIndex
	return logEntry, nil
}

func (le *LogEntry) EncodeLogEntry() ([]byte, error){

	// marshal pbEntry with protobuf
	encodedLogEntry, err := proto.Marshal(le.Entry)
	if err != nil {
		return nil, err
	}
	return encodedLogEntry, nil 
}

func (le *LogEntry) DecodeLogEntry(encodedLogEntry []byte) error {

	// decode log Entry with protobuf
	pbEntry := new(protobuf.Entry)
	err := proto.Unmarshal(encodedLogEntry, pbEntry)
	if err != nil{
		return err
	}

	le.Entry = pbEntry

	// process fast index
	command := NewCommandFromRaw(pbEntry.GetCommandName(), pbEntry.GetCommandContent())
	if command == nil {
		return LogEntryContentDecodeError
	}
	le.fastIndex, err = command.GetFastIndex()
	if err != nil {
		return LogEntryFastIndexError
	}
    return nil

}

// Encodes the log Entry to a buffer. Returns the number of bytes
// written and any error that may have occurred.
// Warning: the return length does not count the length characters and "\n" !!!!!!
func (le *LogEntry) LogStore(writer io.Writer) (int, error) {

	// encode with protobuf
	encodedLogEntry, err := le.EncodeLogEntry()
	if err != nil {
		return -1, err
	}

	// write the length of the log Entry, format []byte (protobuf encoded)
	if _, err = fmt.Fprintf(writer, "%8x\n", len(encodedLogEntry)); err != nil {
		return -1, err
	}

	// return the actual written []byte length
	return writer.Write(encodedLogEntry)
}

// Decodes the log Entry from a buffer. Returns the number of bytes read and
// any error that occurs.
func (le *LogEntry) LogReload(reader io.Reader) (int, error) {

	// read from the buffer the length of byte array
	var length int
	_, err := fmt.Fscanf(reader, "%8x\n", &length)
	if err != nil {
		return -1, err
	}

	// data buffer for read []byte
	byteLogEntry := make([]byte, length)
	// read encoded data
	_, err = io.ReadFull(reader, byteLogEntry)
	if err != nil {
		return -1, err
	}

	// decode from []byte, with protobuf
	if err = le.DecodeLogEntry(byteLogEntry); err != nil {
		return -1, err
	}

	return length + 8 + 1, nil
}

func (le *LogEntry) GetFastIndex() string{
	return le.fastIndex
}
