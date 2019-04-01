package glog

import (
	"fmt"
	"testing"
)

func TestGetLogger(t *testing.T) {

	logger := GetLogger()
	levels := logger.ShowLevel()
	fmt.Printf("%+v\n", levels)
	logger.SetGlobalLevel(ERROR)
	levels = logger.ShowLevel()
	fmt.Printf("%+v\n", levels)

	logger.Info("info msg")
	logger.Error("error msg")
	logger.Fatalf("fatal msg, %s", t.Name())
}

func TestNewLogger(t *testing.T) {
	console := NewConsoleAdapter(ERROR, false, true)
	file := NewFileAdapter(INFO, ".", "test.log")
	logger := NewLogger(DashMillisecondFormat, true,
		console, file)

	fmt.Printf("%+v\n", logger.ShowLevel())
	logger.Fatal("fatal msg")
	logger.Error("error msg")
	logger.Info("info msg")
}
