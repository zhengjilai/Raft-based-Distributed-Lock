package node

import (
	"errors"
	"fmt"
	"github.com/dlock_raft/storage"
	"github.com/dlock_raft/utils"
	"os"
	"sync"
	"time"
)

const (
	// the fixed path for yaml format config file
	ConfigYamlFilePath = "../config/config.yaml"
)

var ReadConfigYamlError = errors.New("dlock_raft.init_node: Read yaml config error")
var ConstructLoggerError = errors.New("dlock_raft.init_node: Construct logger error")

type Node struct{

	// the instance for node config, read from config.yaml
	NodeConfigInstance *NodeConfig

	// the node log handler
	NodeLogger *utils.Logger

	// the node context
	NodeContextInstance *NodeContext

	// the in-memory state maps
	StateMapKVStore *storage.StateMapMemoryKVStore
	StateMapDLock *storage.StateMapMemoryDLock

	// the in-memory logEntry
	LogEntryInMemory *storage.LogMemory

	// all peers
	PeerList []*PeerNode

	// mutex for node object
	mutex *sync.RWMutex
}

type NodeOperators interface {

	BecomeFollower(term uint64) error
	BecomeLeader(term uint64)

}

func NewNode() (*Node, error){

	// read node config from yaml file
	nodeConfigInstance, err := NewNodeConfigFromYaml(ConfigYamlFilePath)
	if err != nil{
		return nil, ReadConfigYamlError
	}

	// new log object, with log path read in node config
	currentTimeString := time.Now().Format("20060102-150405")
	logFileName := nodeConfigInstance.Storage.LogPath + "Dlock-" + currentTimeString + ".log"
	logFileHandler, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return nil, ConstructLoggerError
	}
	nodeLoggerInstance, err := utils.New("DLock-Raft-Node", 1, logFileHandler)
	if err != nil {
		fmt.Println(err)
		return nil, ConstructLoggerError
	}

	// construct a new node object
	node := new(Node)
	node.NodeConfigInstance = nodeConfigInstance
	node.NodeLogger = nodeLoggerInstance

	return node, nil
}

