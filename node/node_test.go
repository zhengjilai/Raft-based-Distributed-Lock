package node

import (
	"fmt"
	"testing"
)

func TestNewNode(t *testing.T){

	nodeTest, err := NewNode()
	if err != nil {
		t.Error(fmt.Sprintf("Error happens when creating a new node: %s", err))
	}

	nodeTest.nodeLogger.Debug("This is Debug!")
	nodeTest.nodeLogger.DebugF("Here are some numbers: %d %d %f", 10, -3, 3.14)
	// Give the Warning
	nodeTest.nodeLogger.Warning("This is Warning!")
	nodeTest.nodeLogger.WarningF("This is Warning!")
	// Show the error
	nodeTest.nodeLogger.Error("This is Error!")
	nodeTest.nodeLogger.ErrorF("This is Error!")
	// Notice
	nodeTest.nodeLogger.Notice("This is Notice!")
	nodeTest.nodeLogger.NoticeF("%s %s", "This", "is Notice!")
	// Show the info
	nodeTest.nodeLogger.Info("This is Info!")
	nodeTest.nodeLogger.InfoF("This is %s!", "Info")

	t.Log(fmt.Println("Solution of the LinearEquationSystem: ", nodeTest))
}