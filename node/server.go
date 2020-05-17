package node

import (
	"github.com/dlock_raft/utils/logger"
	"errors"
	"os"
	"fmt"
	"time"
)

const (
	// the fixed path for yaml format config file
	ConfigYamlFilePath = "../config/config.yaml"
)

var ReadConfigYamlError = errors.New("Dlock_raft.init_server: Read yaml config error.")
var ConstructLoggerError = errors.New("Dlock_raft.init_server: Construct logger error.")

type Server struct{

	// the instance for server config, read from config.yaml
	serverConfig *ServerConfig

	// the server log handler
	serverLogger *logger.Logger
}

func NewServer() (*Server, error){

	// read server config from yaml file
	serverConfigInstance, err := NewServerConfigFromYaml(ConfigYamlFilePath)
	if err != nil{
		return nil, ReadConfigYamlError
	}

	// new log object, with log path read in server config
	currentTimeString := time.Now().Format("20060102-150405")
	logFileName := serverConfigInstance.Storage.LogPath + "Dlock-" + currentTimeString + ".log"
	logFileHandler, err := os.OpenFile(logFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err != nil {
		fmt.Println(err)
		return nil, ConstructLoggerError
	}
	serverLoggerInstance, err := logger.New("DLock-Raft-Server", 1, logFileHandler)
	if err != nil {
		fmt.Println(err)
		return nil, ConstructLoggerError
	}

	// construct a new server object
	server := new(Server)
	server.serverConfig = serverConfigInstance
	server.serverLogger = serverLoggerInstance

	fmt.Println(server)

	return server, nil
}

