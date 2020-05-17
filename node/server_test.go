package node

import (
	"fmt"
	"testing"
)

func TestNewServer(t *testing.T){

	serverTest, err := NewServer()
	if err != nil {
		t.Error(fmt.Sprintf("Error happens when creating a new server: %s", err))
	}

	serverTest.serverLogger.Debug("This is Debug!")
	serverTest.serverLogger.DebugF("Here are some numbers: %d %d %f", 10, -3, 3.14)
	// Give the Warning
	serverTest.serverLogger.Warning("This is Warning!")
	serverTest.serverLogger.WarningF("This is Warning!")
	// Show the error
	serverTest.serverLogger.Error("This is Error!")
	serverTest.serverLogger.ErrorF("This is Error!")
	// Notice
	serverTest.serverLogger.Notice("This is Notice!")
	serverTest.serverLogger.NoticeF("%s %s", "This", "is Notice!")
	// Show the info
	serverTest.serverLogger.Info("This is Info!")
	serverTest.serverLogger.InfoF("This is %s!", "Info")

	t.Log(fmt.Println("Solution of the LinearEquationSystem: ", serverTest))
}