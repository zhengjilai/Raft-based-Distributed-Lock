package storage

import (
	"fmt"
	"github.com/dlock_raft/protobuf"
	"google.golang.org/protobuf/proto"
	"io"
)

// A log entry type for storing, encoding and decoding entries
type LogEntry struct {
	entry *protobuf.Entry
}

func NewLogEntry(term uint64, index uint64, command CommandOperator) (*LogEntry){

	// elements are stored in protobuf.Entry
	pbEntry := new(protobuf.Entry)
	pbEntry.Term = term
	pbEntry.Index = index
	pbEntry.CommandName = command.GetCommandName()
	pbEntry.CommandContent = command.GetCommandContent()

	// the feedback LogEntry entity
	logEntry := new(LogEntry)
	logEntry.entry = pbEntry

    return logEntry
}

func (le *LogEntry) EncodeLogEntry() ([]byte, error){

	// marshal pbEntry with protobuf
	encodedLogEntry, err := proto.Marshal(le.entry)
	if err != nil {
		return nil, err
	}
	return encodedLogEntry, nil 
}

func (le *LogEntry) DecodeLogEntry(encodedLogEntry []byte) (error){

	// decode log entry with protobuf
	pbEntry := new(protobuf.Entry)
	err := proto.Unmarshal(encodedLogEntry, pbEntry)
	if err != nil{
		return err
	}

	le.entry = pbEntry
    return nil

}

// Encodes the log entry to a buffer. Returns the number of bytes
// written and any error that may have occurred.
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

// Decodes the log entry from a buffer. Returns the number of bytes read and
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
