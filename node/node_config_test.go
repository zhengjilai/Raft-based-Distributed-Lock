package node

import (
	"fmt"
	"testing"
)

func TestNodeConfig(t *testing.T){
	// read node config from yaml file
	nodeConfigInstance, err := NewNodeConfigFromYaml("../config/test_configs/config-node-test.yaml")
	if err != nil{
		t.Error(fmt.Sprintf("Error happens when creating a new node config from yaml: %s", err))
	}

	t.Log(fmt.Println("Node config :\n", nodeConfigInstance))
}


