// the interchange before dlock command is appended into LogMemory
package node

type DlockInterchange struct {

	// the NodeRef Object
	NodeRef *Node

	// the map for pending dlock requests
	PendingAcquire map[string][]byte
}
