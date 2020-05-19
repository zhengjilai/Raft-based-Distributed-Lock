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

	nodeTest.NodeLogger.Debug("This is Debug!")
	nodeTest.NodeLogger.DebugF("Here are some numbers: %d %d %f", 10, -3, 3.14)
	// Give the Warning
	nodeTest.NodeLogger.Warning("This is Warning!")
	nodeTest.NodeLogger.WarningF("This is Warning!")
	// Show the error
	nodeTest.NodeLogger.Error("This is Error!")
	nodeTest.NodeLogger.ErrorF("This is Error!")
	// Notice
	nodeTest.NodeLogger.Notice("This is Notice!")
	nodeTest.NodeLogger.NoticeF("%s %s", "This", "is Notice!")
	// Show the info
	nodeTest.NodeLogger.Info("This is Info!")
	nodeTest.NodeLogger.InfoF("This is %s!", "Info")

	t.Log(fmt.Println("Solution of the LinearEquationSystem: ", nodeTest))
}