// This file is reserved for implementing commands
// We currently implement two commands
// 1: Key Value Storage (CommandName: KVStore)
// 2: Distributed Lock (CommandName: DLock)

// You can edit this file to add new commands for Raft
// as long as your command satisfies the interface standard

package storage

import (
	"encoding/json"
)

// inherited from Command
// always has the default CommandName as "KVStore"
type CommandKVStore struct {
	Command
}

type KeyValue struct {
	// Key should be unique
	Key string `json:"key"`
	Value []byte `json:"value"`
}

func NewCommandKVStore(key string, value []byte) (*CommandKVStore, error) {
	
	// key value object pending to be marshalled
	keyValue := new(KeyValue)
	keyValue.Key = key
	keyValue.Value = value

	// marshal the kv object
	encodedKeyValue, err := json.Marshal(keyValue)
	if err != nil {
		return nil, err
	}

	// construct the command object
	commandKVStore := new(CommandKVStore)
	// note that "KVStore" is predetermined for CommandKVStore
	commandKVStore.SetCommandName("KVStore")
	commandKVStore.SetCommandContent(encodedKeyValue)
	// KVStore interface
	commandKVStore.commandOperator = commandKVStore

	return commandKVStore, nil
}

func (c *CommandKVStore) SetAsKVStore(keyValue *KeyValue) error {

	// marshal the kv object
	encodedKeyValue, err := json.Marshal(keyValue)
	if err != nil {
		return err
	}

	// construct the command object
	commandKVStore := new(CommandKVStore)
	// note that "KVStore" is predetermined for CommandKVStore
	commandKVStore.SetCommandContent(encodedKeyValue)

	return nil
}

func (c *CommandKVStore) GetAsKVStore() (*KeyValue, error){
	
	// unmarshal the kv object
	decodedKeyValue := new(KeyValue)
	err := json.Unmarshal(c.GetCommandContent(), &decodedKeyValue)
	if err != nil {
		return nil, err
	}

	return decodedKeyValue, nil
}

// the fast index interface specific to KVStore
func (c *CommandKVStore) GetFastIndex() (string, error) {
	kvStore, err := c.GetAsKVStore()
	if err != nil {
		return "", err
	}
	// the fast index is set as key
	return "KVStore:" + kvStore.Key, nil
}


// inherited from Command
// always has the default CommandName as "DLock"
type CommandDLock struct {
	Command
}

// OrigOwner indicates the original owner of this lock
// NewOwner indicates the current owner of this lock (after lock switch)
// Timestamp is generated by time.Now().UnixNano(), by user. 
type LockInfo struct {
	// LockId should be unique
	LockId uint32 `json:"lock_id"`
	// LockName should also be unique, will be used as fastIndex
	LockName string `json:"lock_name"`
	OrigOwner string `json:"orig_owner"`
	NewOwner string `json:"new_owner"`
	Timestamp int64 `json:"timestamp"`
}

func NewCommandDLock(lockId uint32, lockName string, origOwner string,
									newOwner string, timestamp int64) (*CommandDLock, error) {
	
	// dlock object object pending to be marshalled
	lockInfo := new(LockInfo)
	lockInfo.LockId = lockId
	lockInfo.LockName = lockName
	lockInfo.OrigOwner = origOwner
	lockInfo.NewOwner = newOwner
	lockInfo.Timestamp = timestamp

	// marshal the dlock object
	encodedLockInfo, err := json.Marshal(lockInfo)
	if err != nil {
		return nil, err
	}

	// construct the command object
	commandDLock := new(CommandDLock)
	// note that ""DLock" is predetermined for CommandDLock
	commandDLock.SetCommandName("DLock")
	commandDLock.SetCommandContent(encodedLockInfo)
	commandDLock.commandOperator = commandDLock

	return commandDLock, nil
}

func (c *CommandDLock) SetAsDLockInfo(lockInfo *LockInfo) error {

	// marshal the dlock object
	encodedLockInfo, err := json.Marshal(lockInfo)
	if err != nil {
		return err
	}

	// construct the command object
	commandDLock := new(CommandDLock)
	// note that ""DLock" is predetermined for CommandDLock
	commandDLock.SetCommandContent(encodedLockInfo)

	return nil
}

func (c *CommandDLock) GetAsDLockInfo() (*LockInfo, error){
	
	// unmarshal the kv object
	decodedLockInfo := new(LockInfo)
	err := json.Unmarshal(c.GetCommandContent(), decodedLockInfo)
	if err != nil {
		return nil, err
	}

	return decodedLockInfo, nil
}

// the fast index interface specific to DLockInfo
func (c *CommandDLock) GetFastIndex() (string, error) {
	dLockInfo, err := c.GetAsDLockInfo()
	if err != nil {
		return "", err
	}
	// the fast index is set as LockName
	return "DLock:" + dLockInfo.LockName, nil
}