// an implementation for NodeConfig
// NodeConfig are read from config.yaml
// all information in NodeConfig is static
package node

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"reflect"
)

var nodeConfigNilValueError = errors.New("dlock_raft.node_config: " +
	"nil value occurs in Node config")
var nodeConfigIdZeroError = errors.New("dlock_raft.node_config: " +
	"0 should never be used as an id")
var nodeConfigPeerNumberError = errors.New("dlock_raft.node_config: " +
	"number of nodes in peer_id, peer_address and peer_cli_address should be equal")

type NodeConfig struct{

	// int32 id for node
	Id struct {
		// self node id, int32
		SelfId uint32 `yaml:"self_id"`
		// peer node ids, int32
		PeerId []uint32 `yaml:"peer_id"`
	} `yaml:"id"`

	Network struct {
		// self node address, ip:port
		SelfAddress string `yaml:"self_address"`
		// self cli address, ip:port
		SelfCliAddress string `yaml:"self_cli_address"`
		// peer node addresses, ip:port
		PeerAddress []string `yaml:"peer_address"`
		// peer node cli-srv addresses, ip:port
		PeerCliAddress []string `yaml:"peer_cli_address"`
	} `yaml:"network"`

    Parameters struct {

		// the interval for leader to push heart beat package to followers, unit:ms
		HeartBeatInterval uint32 `yaml:"heart_beat_interval"`

		// the timeout for every AppendEntriesRequest, unit:ms
		AppendEntriesTimeout uint32 `yaml:"append_entries_timeout"`

		// the maximum and minimum time before become a candidate, unit:ms
		MinWaitTimeCandidate uint32 `yaml:"min_wait_time_candidate"`
		MaxWaitTimeCandidate uint32 `yaml:"max_wait_time_candidate"`
		
		// maximum number of log units in RecoverEntriesResponse and AppendEntryRequest
		MaxLogUnitsRecover uint32 `yaml:"max_log_units_recover"`

		// the interval for log backup
		LogBackupInterval uint32 `yaml:"log_back_up_interval"`

		// the interval for polling
		PollingInterval uint32 `yaml:"polling_interval"`
		// the timeout for state change (wait to be committed)
		StateChangeTimeout uint32 `yaml:"state_change_timeout"`

		// the default expire for acquirement in pending list
		AcquirementExpire uint32 `yaml:"acquirement_expire"`

	} `yaml:"parameters"`

    Storage struct {
		// the path for log file
		LogPath string `yaml:"log_path"`
		
		// the path for persistent entry storage
		EntryStoragePath string `yaml:"entry_storage_path"`
	}
}

func NewNodeConfigFromYaml(configPath string) (*NodeConfig, error) {
	
	// read config file as []byte
	fileData, err := ioutil.ReadFile(configPath)
	if err != nil{
		return nil, err
	}

	// the config of yaml format
	nodeConfig := new(NodeConfig)

	// unmarshall the config byte and return
	err = yaml.Unmarshal(fileData, &nodeConfig)
	if err != nil{
		return nil, err
	}

	// test nil values
	if nodeConfig.Id.PeerId == nil ||
		nodeConfig.Network.PeerAddress == nil ||
		nodeConfig.Network.PeerCliAddress == nil ||
		nodeConfig.Storage.LogPath == "" ||
		nodeConfig.Storage.EntryStoragePath == "" ||
		nodeConfig.Network.SelfCliAddress == "" ||
		nodeConfig.Network.SelfAddress == "" {
		return nil, nodeConfigNilValueError
	}

	// test the number of peers in every node config item
	if len(nodeConfig.Id.PeerId) != len(nodeConfig.Network.PeerAddress) ||
		len(nodeConfig.Id.PeerId) != len(nodeConfig.Network.PeerCliAddress) {
		return nil, nodeConfigPeerNumberError
	}

	// test zero in nodeIds
	if nodeConfig.Id.SelfId == 0 {
		return nil, nodeConfigIdZeroError
	}
	for _, item := range nodeConfig.Id.PeerId {
		if item == 0 {
			return nil, nodeConfigIdZeroError
		}
	}

	// set default values
	var defaultValue = map[string]interface{}{
		"heart_beat_interval": 50,
		"append_entries_timeout": 150,
		"min_wait_time_candidate": 150,
		"max_wait_time_candidate": 300,
		"max_log_units_recover": 200,
		"log_back_up_interval": 1000,
		"polling_interval": 20,
		"state_change_timeout": 2000,
		"acquirement_expire": 500,
	}

	err2 := structByReflect(defaultValue, &nodeConfig.Parameters)
	if err2 != nil {
		return nil, err2
	}

	return nodeConfig, nil
}

func structByReflect(data map[string]interface{}, inStructPtr interface{}) error {
	rType := reflect.TypeOf(inStructPtr)
	rVal := reflect.ValueOf(inStructPtr)
	if rType.Kind() == reflect.Ptr {
		rType = rType.Elem()
		rVal = rVal.Elem()
	} else {
		return errors.New("dlock_raft.node_config: inStructPtr must be ptr to struct")
	}
	// scan the struct
	for i := 0; i < rType.NumField(); i++ {
		t := rType.Field(i)
		f := rVal.Field(i)

		// continue to set default value only if current value is not 0
		if f.Uint() != 0 {
			continue
		}

		key := t.Tag.Get("yaml")
		if v, ok := data[key]; ok {
			dataType := reflect.TypeOf(v)
			structType := f.Type()
			if structType == dataType {
				fmt.Println(structType)
				f.Set(reflect.ValueOf(v))
			} else {
				if dataType.ConvertibleTo(structType) {
					f.Set(reflect.ValueOf(v).Convert(structType))
				} else {
					return errors.New("dlock_raft.node_config: " + t.Name + " type mismatch")
				}
			}
		} else {
			return errors.New("dlock_raft.node_config: " + t.Name + " not found")
		}
	}
	return nil
}
