package main

import (
	"fmt"
	"github.com/dlock_raft/node"
	"time"
)

func main()  {
	nodeInstance, err := node.NewNode()
	if err != nil {
		fmt.Printf("Error happens when starting a node, error: %s\n", err)
	}
	nodeInstance.InitRaftConsensusModule()

	// exit main goroutine normally only if a Stop signal is received
	select {
		case _, ok := <-nodeInstance.NodeContextInstance.StopChan:
			if ok {
				nodeInstance.NodeLogger.Infof("Normal exit.")
			} else {
				nodeInstance.NodeLogger.Infof("Abnormal exit.")
			}
	}
	nodeInstance.NodeLogger.Infof("Node Exit at %s", time.Now())
}
