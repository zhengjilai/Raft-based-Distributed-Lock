// server state struct
// code revised from https://github.com/goraft/raft
package node

// ServerContextInterface represents the current state of the server.
type ServerContextInterface interface {
	CurrentTerm() uint64
	CurrentIndex() uint64
	CommitIndex() uint64
}

// ServerContext is the concrete implementation of ServerContext.
type ServerContext struct {
	CurrentIndex uint64
	CurrentTerm  uint64
	CommitIndex  uint64
}

// CurrentTerm returns current term the server is in.
func (c *ServerContext) GetCurrentTerm() uint64 {
	return c.CurrentTerm
}

// CurrentIndex returns current index the server is at.
func (c *ServerContext) GetCurrentIndex() uint64 {
	return c.CurrentIndex
}

// CommitIndex returns last commit index the server is at.
func (c *ServerContext) GetCommitIndex() uint64 {
	return c.CommitIndex
}

// Init an ServerContext instance
func NewServerContext(term uint64, index uint64, commit uint64) *ServerContext{

	// construct the ServerContext object and return
	serverContext := new(ServerContext)
	serverContext.CurrentTerm = term
	serverContext.CurrentIndex = index
	serverContext.CommitIndex = commit

	return serverContext
}
