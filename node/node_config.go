// an implementation for NodeConfig
// NodeConfig are read from config.yaml
// all information in NodeConfig is static
package node

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

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
		// peer node addresses, ip:port
		PeerAddress []string `yaml:"peer_address"`
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

	} `yaml:"parameters"`

    Storage struct {
		// the path for log file
		LogPath string `yaml:"log_path"`
		
		// the path for persistant entry storage
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
	return nodeConfig, nil
}

