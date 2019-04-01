package glog

import (
	"testing"
)

func TestFileLines(t *testing.T) {
	var want int64 = 3
	fileLine, _ := FileLines("hello.txt")
	if fileLine != want{
		t.Errorf("wanted : %d, actual: %d",want,fileLine)
	}

}

