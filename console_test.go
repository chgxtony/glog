package glog

import (
	"testing"
	"time"
)

func TestWriteLogmsg(t *testing.T) {
	console := NewConsoleAdapter(INFO, true, false)

	console.Write(&loggerMsg{
		Itime:  time.Now().Format(DashMillisecondFormat),
		Ilevel: INFO,
		File:   "test.go",
		Line:   17,
		Body:   "hello world",
	})

}
