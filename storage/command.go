package storage

type Command struct {
	
	// command name, format string
	commandName string

	// command content, may have been encoded
	commandContent []byte

	// required interface for command
	CommandOp CommandOperator

}

type CommandOperator interface{

    GetCommandName() string
	GetContent() []byte

}

func (c *Command) GetCommandName() (string) {	
	return c.commandName
}

func (c *Command) GetCommandContent() ([]byte) {	
	return c.commandContent
}

func (c *Command) SetCommandName(commandName string){	
	c.commandName = commandName
	return 
}

func (c *Command) SetCommandContent(commandContent []byte) {	
	c.commandContent = commandContent
	return
}