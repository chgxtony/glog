package glog

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"sync"
	"time"
)

const FILE_ADAPTER_NAME = "file"

type UNIT int64
type ROLLTYPE int
type SliceDateType int

const (
	FILE_SLICE_DATE_NULL SliceDateType = iota
	FILE_SLICE_DATE_YEAR
	FILE_SLICE_DATE_MONTH
	FILE_SLICE_DATE_DAY
	FILE_SLICE_DATE_HOUR
)

const (
	KB UNIT = 1 << (iota * 10) // 2^0, 2^10, 2^20, 2^30
	MB
	GB
	TB
)

const (
	RollingDaily ROLLTYPE = iota
	RollingFileSize
	RollingFileLine
)

// file fileWriter
type FileWriter struct {
	lock      sync.RWMutex
	writer    *os.File
	startLine int64
	startTime int64
	logfile   string //log file , absolute path
}

//slice file by date (y, m, d, h), rename file is file_time.log and recreate file
func (fw *FileWriter) sliceByDate(dateSlice SliceDateType) error {

	filename := fw.logfile
	filenameSuffix := path.Ext(filename)
	startTime := time.Unix(fw.startTime, 0)
	nowTime := time.Now()

	oldFilename := ""
	isHaveSlice := false
	if (dateSlice == FILE_SLICE_DATE_YEAR) &&
		(startTime.Year() != nowTime.Year()) {
		isHaveSlice = true
		oldFilename = strings.Replace(filename, filenameSuffix, "", 1) + "_" + startTime.Format("2006") + filenameSuffix
	}
	if (dateSlice == FILE_SLICE_DATE_MONTH) &&
		(startTime.Format("200601") != nowTime.Format("200601")) {
		isHaveSlice = true
		oldFilename = strings.Replace(filename, filenameSuffix, "", 1) + "_" + startTime.Format("200601") + filenameSuffix
	}
	if (dateSlice == FILE_SLICE_DATE_DAY) &&
		(startTime.Format("20060102") != nowTime.Format("20060102")) {
		isHaveSlice = true
		oldFilename = strings.Replace(filename, filenameSuffix, "", 1) + "_" + startTime.Format("20060102") + filenameSuffix
	}
	if (dateSlice == FILE_SLICE_DATE_HOUR) &&
		(startTime.Format("2006010215") != startTime.Format("2006010215")) {
		isHaveSlice = true
		oldFilename = strings.Replace(filename, filenameSuffix, "", 1) + "_" + startTime.Format("2006010215") + filenameSuffix
	}

	if isHaveSlice == true {
		//close file handler
		fw.writer.Close()
		err := os.Rename(fw.logfile, oldFilename)
		if err != nil {
			return err
		}
		err = fw.initFile()
		if err != nil {
			return err
		}
	}

	return nil
}

//slice file by size, if maxSize < fileSize, rename file is file_size_maxSize_time.log and recreate file
func (fw *FileWriter) sliceByFileSize(maxSize int64) error {

	filename := fw.logfile
	filenameSuffix := path.Ext(filename)
	nowSize, _ := fw.getFileSize(filename)

	if nowSize >= maxSize {
		//close file handle
		fw.writer.Close()
		timeFlag := time.Now().Format("2006-01-02-15.04.05.9999")
		oldFilename := strings.Replace(filename, filenameSuffix, "", 1) + "." + timeFlag + filenameSuffix
		err := os.Rename(filename, oldFilename)
		if err != nil {
			return err
		}
		err = fw.initFile()
		if err != nil {
			return err
		}
	}

	return nil
}

//slice file by line, if maxLine < fileLine, rename file is file_line_maxLine_time.log and recreate file
func (fw *FileWriter) sliceByFileLine(maxLine int64) error {

	filename := fw.logfile
	filenameSuffix := path.Ext(filename)
	startLine := fw.startLine

	if startLine >= maxLine {
		//close file handle
		fw.writer.Close()
		timeFlag := time.Now().Format("2006-01-02-15.04.05.9999")
		oldFilename := strings.Replace(filename, filenameSuffix, "", 1) + "." + timeFlag + filenameSuffix
		err := os.Rename(filename, oldFilename)
		if err != nil {
			return err
		}
		err = fw.initFile()
		if err != nil {
			return err
		}
	}

	return nil
}

func (fw *FileWriter) flush() {
	fw.writer.Close()
}

// init file
func (fw *FileWriter) initFile() error {

	//check file exits, otherwise create a file
	fp, err := OpenOrCreateFile(fw.logfile)

	if err != nil {
		return err
	}
	fw.writer = fp

	// get start time
	fw.startTime = time.Now().Unix()

	// get file start lines
	nowLines, err := FileLines(fw.logfile)
	if err != nil {
		return err
	}
	fw.startLine = nowLines

	return nil
}

//get file size
//params : logfile
//return : fileSize(byte int64), error
func (fw *FileWriter) getFileSize(filename string) (int64, error) {
	fileInfo, err := os.Stat(filename)
	if err != nil {
		return 0, err
	}

	return fileInfo.Size() / 1024, nil
}

// writers by config
func (fw *FileWriter) writeByConfig(config *FileConfig, loggerMsg *loggerMsg) error {

	fw.lock.Lock()
	defer fw.lock.Unlock()

	if config.RollingType == RollingDaily {
		// file slice by date
		err := fw.sliceByDate(config.DateSlice)
		if err != nil {
			return err
		}
	}
	if config.RollingType == RollingFileLine {
		// file slice by line
		err := fw.sliceByFileLine(config.MaxLine)
		if err != nil {
			return err
		}
	}
	if config.RollingType == RollingFileSize {
		// file slice by size
		err := fw.sliceByFileSize(config.MaxSize)
		if err != nil {
			return err
		}
	}

	msg := ""
	if config.JsonFlag == true {
		jsonByte, _ := json.Marshal(loggerMsg)
		msg = string(jsonByte) + "\n"
	} else {
		msg = formatLoggerMsg(loggerMsg) + "\n"
	}

	if config.Level() <= loggerMsg.Ilevel {
		fw.writer.Write([]byte(msg))
		if config.MaxLine != 0 {
			if config.JsonFlag == true {
				fw.startLine += 1
			} else {
				fw.startLine += int64(strings.Count(msg, "\n"))
			}
		}
	}

	return nil
}

type FileConfig struct {
	// use json format to output
	JsonFlag bool

	LogLevel LOGLEVEL

	// log store dir
	FilePath string

	// log logfile
	Filename string

	RollingType ROLLTYPE

	// max file size
	MaxSize int64

	// max file line
	MaxLine int64

	// file slice by date
	// "y" Log files are cut through year
	// "m" Log files are cut through mouth
	// "d" Log files are cut through day
	// "h" Log files are cut through hour
	DateSlice SliceDateType

	LoggerConfig
}

func (config *FileConfig) Level() LOGLEVEL {
	return config.LogLevel
}

func (config *FileConfig) SetLevel(loglevel LOGLEVEL) {
	config.LogLevel = loglevel
}

func (config *FileConfig) IsJson() bool {
	return config.JsonFlag
}

func (config *FileConfig) CheckConfig() error {
	if config.FilePath == "" || config.Filename == "" {
		return errors.New("config FilePath and Filename can't be empty")
	}

	switch config.RollingType {
	case RollingDaily:
		if config.DateSlice == FILE_SLICE_DATE_NULL {
			return errors.New("when RollingType is RollingDaily, must config DateSlice")
		}
	case RollingFileSize:
		if config.MaxSize == 0 {
			return errors.New("when RollingType is RollingFileSize, must config MaxSize")
		}

	case RollingFileLine:
		if config.MaxLine == 0 {
			return errors.New("when RollingType is RollingFileLine, must config MaxLine")
		}
	default:
		return errors.New("must config RollingType")
	}
	return nil
}

// adapter file
type FileAdapter struct {
	fileWriter *FileWriter
	FileConfig
	AdapterLogger
}

func (*FileAdapter) Name() string {
	return FILE_ADAPTER_NAME
}

func (adapterFile *FileAdapter) Init() error {
	adapterFile.CheckConfig()
	fmt.Printf("[GLOG] > [%s adapter] init success\n", adapterFile.Name())
	return nil
}

// Write
func (adapterFile *FileAdapter) Write(loggerMsg *loggerMsg) error {

	var accessChan = make(chan error, 1)

	go func() {
		err := adapterFile.fileWriter.writeByConfig(&adapterFile.FileConfig, loggerMsg)
		if err != nil {
			accessChan <- err
			return
		}
		accessChan <- nil
	}()

	var accessErr error
	accessErr = <-accessChan

	if accessErr != nil {
		return accessErr.(error)
	}
	return nil
}

// Flush
func (adapterFile *FileAdapter) Flush() {
	adapterFile.fileWriter.flush()
}

func NewFileWriter(filepath, filename string) *FileWriter {
	fw := &FileWriter{
		logfile: path.Join(filepath, filename),
	}
	fw.initFile()
	return fw

}

func NewFileAdapter(loglevel LOGLEVEL, filepath string, filename string) AbstractLogger {
	fileConfig := FileConfig{
		FilePath:    filepath,
		Filename:    filename,
		JsonFlag:    false,
		LogLevel:    loglevel,
		RollingType: RollingDaily,
		DateSlice:   FILE_SLICE_DATE_DAY,
	}
	err := fileConfig.CheckConfig()
	if err != nil {
		printError("file config illegal : %s", err.Error())
	}

	fileWriter := NewFileWriter(fileConfig.FilePath, fileConfig.Filename)
	return &FileAdapter{
		fileWriter: fileWriter,
		FileConfig: fileConfig,
		AdapterLogger: AdapterLogger{
			Id: "defaultFile",
		},
	}
}

func init() {
	fileAdapter := func() AbstractLogger {
		return &FileAdapter{}
		//return NewFileAdapter(INFO, ".", "test.log")
	}
	Register(FILE_ADAPTER_NAME, fileAdapter)
}
