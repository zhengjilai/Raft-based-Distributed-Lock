package node

import (
	"fmt"
	"testing"
)

func TestNodeContext(t *testing.T){

	// Node Context predetermined
	testTerm := uint64(4000)
	testIndex := uint64(9225)
	testCommit := uint64(14005)

	// New node context
	testNodeContext := NewNodeContext(testTerm, testIndex, testCommit)

	t.Log(fmt.Printf("Node context: %d, %d, %d \n",
		testNodeContext.CurrentTerm, testNodeContext.CurrentIndex, testNodeContext.CommitIndex))
}
