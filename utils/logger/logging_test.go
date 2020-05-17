package logger

import (
	"fmt"
	"os"
	"testing"
)

func TestServerLoggerModule(t *testing.T){

	// new log object, with log path read in server config
	testLogFileName := "../../var/logger_test.log"
	// log handler, format io.writer
	logFileHandler, err1 := os.OpenFile(testLogFileName, os.O_RDWR|os.O_CREATE, 0644)
	if err1 != nil {
		t.Error(fmt.Sprintf("Error happens when opening a test logger file: %s\n", err1))
	}

	// the test logger instance
	serverLoggerInstance, err2 := New("Test-DLock-Raft-Server", 1, logFileHandler)
	if err2 != nil {
		t.Error(fmt.Sprintf("Error happens when creating a test logger handler: %s\n", err2))
	}
	// set log level as DEBUG
	serverLoggerInstance.SetLogLevel(DebugLevel)

	// Give the Debug
	serverLoggerInstance.Debug("This is Debug!")
	serverLoggerInstance.DebugF("Here are some numbers: %d %d %f", 10, -3, 3.14)
	// Give the Warning
	serverLoggerInstance.Warning("This is Warning!")
	serverLoggerInstance.WarningF("This is Warning!")
	// Show the error
	serverLoggerInstance.Error("This is Error!")
	serverLoggerInstance.ErrorF("This is Error!")
	// Notice
	serverLoggerInstance.Notice("This is Notice!")
	serverLoggerInstance.NoticeF("%s %s", "This", "is Notice!")
	// Show the info
	serverLoggerInstance.Info("This is Info!")
	serverLoggerInstance.InfoF("This is %s!", "Info")

	t.Log(fmt.Println("Test logger finished, see result in var/logger_test.log"))
}