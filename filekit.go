package glog

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

// create file
func CreateFile(filename string) error {
	f, err := os.Create(filename)
	defer f.Close()
	return err
}

// file or path is exists
func IsExist(path string) bool {
	_, err := os.Stat(path)
	return err == nil || os.IsExist(err)
}

func OpenOrCreateFile(filename string) (*os.File, error) {
	ok := IsExist(filename)
	if ok {
		file, err := os.OpenFile(filename, os.O_RDWR|os.O_APPEND, 0766)
		return file, err
	} else {
		file, err := os.Create(filename)
		return file, err
	}
}

func Md5str(s string) string {
	m := md5.New()
	m.Write([]byte(s))
	return hex.EncodeToString(m.Sum(nil))
}

func FileSize(file string) int64 {
	f, e := os.Stat(file)
	if e != nil {
		fmt.Println(e.Error())
		return 0
	}
	return f.Size()
}

//get file lines
//params : logfile
//return : fileLine, error
func FileLines(filename string) (int64, error) {
	file, err := os.OpenFile(filename, os.O_RDONLY, 0766)
	if err != nil {
		return 0, err
	}
	defer file.Close()

	var fileLine int64 = 0
	r := bufio.NewReader(file)
	for {
		_, err := r.ReadString('\n')
		if err != nil || err == io.EOF {
			break
		}
		fileLine += 1
	}
	return fileLine, nil
}
