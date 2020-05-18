package node

import (
	"fmt"
	"testing"
)

func TestServerContext(t *testing.T){

	// Server Context predetermined
	testTerm := uint64(4000)
	testIndex := uint64(9225)
	testCommit := uint64(14005)

	// New server context
	testServerContext := NewServerContext(testTerm, testIndex, testCommit)

	t.Log(fmt.Printf("Server context: %d, %d, %d \n",
		testServerContext.CurrentTerm, testServerContext.CurrentIndex, testServerContext.CommitIndex))
}
