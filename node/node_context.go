// node state struct
// code revised from https://github.com/goraft/raft
package node

// NodeContextInterface represents the current state of the node.
type NodeContextInterface interface {
	CurrentTerm() uint64
	CurrentIndex() uint64
	CommitIndex() uint64
}

// NodeContext is the concrete implementation of NodeContext.
type NodeContext struct {
	CurrentIndex uint64
	CurrentTerm  uint64
	CommitIndex  uint64
}

// CurrentTerm returns current term the node is in.
func (c *NodeContext) GetCurrentTerm() uint64 {
	return c.CurrentTerm
}

// CurrentIndex returns current index the node is at.
func (c *NodeContext) GetCurrentIndex() uint64 {
	return c.CurrentIndex
}

// CommitIndex returns last commit index the node is at.
func (c *NodeContext) GetCommitIndex() uint64 {
	return c.CommitIndex
}

// Init an NodeContext instance
func NewNodeContext(term uint64, index uint64, commit uint64) *NodeContext {

	// construct the NodeContext object and return
	nodeContext := new(NodeContext)
	nodeContext.CurrentTerm = term
	nodeContext.CurrentIndex = index
	nodeContext.CommitIndex = commit

	return nodeContext
}
