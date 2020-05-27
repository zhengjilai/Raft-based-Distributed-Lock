package dlock_raft

import (
	"fmt"
	"github.com/dlock_raft/node"
)

func main()  {
	nodeInstance, err := node.NewNode()
	if err != nil {
		fmt.Printf("Error happens when starting a node, error: %s\n", err)
	}
	nodeInstance.InitRaftConsensusModule()
}
