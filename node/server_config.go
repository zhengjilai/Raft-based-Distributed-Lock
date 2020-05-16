// an implementation for Server
package node

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

type ServerConfig struct{

	// int32 id for server
	Id int32 `yaml:"id"`

	Network struct {
		// server address, ip:port
		SelfAddress string `yaml:"self_address"`
		// peer addresses, ip:port
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
		
		// maximum number of log units in RecoverEntriesResponse 
		MaxLogUnitsRecover uint32 `yaml:"max_log_units_recover"`
	} `yaml:"parameters"`

    Storage struct {
		// the path for log file
		LogPath string `yaml:"log_path"`
		
		// the path for persistant entry storage
		EntryStoragePath string `yaml:"entry_storage_path"`
	}
}

func NewServerConfigFromYaml(configPath string) (*ServerConfig, error) {
	
	// read config file as []byte
	fileData, err := ioutil.ReadFile(configPath)
	if err != nil{
		return nil, err
	}

	// the config of yaml format
	serverConfig := new(ServerConfig)

	// unmarshall the config byte and return
	err = yaml.Unmarshal(fileData, &serverConfig)
	if err != nil{
		return nil, err
	}
	return serverConfig, nil
}

