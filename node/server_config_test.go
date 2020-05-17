package node

import (
	"fmt"
	"testing"
)

func TestServerConfig(t *testing.T){
	// read server config from yaml file
	serverConfigInstance, err := NewServerConfigFromYaml(ConfigYamlFilePath)
	if err != nil{
		t.Error(fmt.Sprintf("Error happens when creating a new server config from yaml: %s", err))
	}

	t.Log(fmt.Println("Server config :\n", serverConfigInstance))
}


