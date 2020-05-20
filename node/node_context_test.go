package node

import (
	"fmt"
	"testing"
)

func TestNodeContext(t *testing.T){

	testNodeContext := NewStartNodeContext()

	t.Log(fmt.Printf("Node context: %d, %d, %d \n",
		testNodeContext.CurrentTerm, testNodeContext.CurrentIndex, testNodeContext.CommitIndex))
}
