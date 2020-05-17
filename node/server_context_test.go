package node

import (
	"fmt"
	"testing"
)

func TestServerContext(t *testing.T){

	// Server Context predetermined
	testTerm := int64(4000)
	testIndex := int64(9225)
	testCommit := int64(14005)

	// New server context
	testServerContext := NewServerContext(testTerm, testIndex, testCommit)

	t.Log(fmt.Printf("Server context: %d, %d, %d \n",
		testServerContext.CurrentTerm, testServerContext.CurrentIndex, testServerContext.CommitIndex))
}
