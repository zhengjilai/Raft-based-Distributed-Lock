package node

import (
	"fmt"
	"testing"
)

func TestNewNode(t *testing.T){

	nodeTest, err := NewNode("../config/config.yaml")
	if err != nil {
		t.Error(fmt.Sprintf("Error happens when creating a new node: %s", err))
	}

	t.Log(fmt.Println("Constructed DLock Node: ", nodeTest))
}