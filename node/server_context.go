// server state struct
// code revised from https://github.com/goraft/raft
package node

// ServerContextInterface represents the current state of the server.
type ServerContextInterface interface {
	CurrentTerm() int64
	CurrentIndex() int64
	CommitIndex() int64
}

// ServerContext is the concrete implementation of ServerContext.
type ServerContext struct {
	CurrentIndex int64
	CurrentTerm  int64
	CommitIndex  int64
}

// CurrentTerm returns current term the server is in.
func (c *ServerContext) GetCurrentTerm() int64 {
	return c.CurrentTerm
}

// CurrentIndex returns current index the server is at.
func (c *ServerContext) GetCurrentIndex() int64 {
	return c.CurrentIndex
}

// CommitIndex returns last commit index the server is at.
func (c *ServerContext) GetCommitIndex() int64 {
	return c.CommitIndex
}

// Init an ServerContext instance
func NewServerContext(term int64, index int64, commit int64) *ServerContext{

	// construct the ServerContext object and return
	serverContext := new(ServerContext)
	serverContext.CurrentTerm = term
	serverContext.CurrentIndex = index
	serverContext.CommitIndex = commit

	return serverContext
}
