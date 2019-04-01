package glog

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"sync"
	"time"
)



type LOGLEVEL int

func (logLevel LOGLEVEL) LevelString() string {
	return levelStringMap[logLevel]
}

func (logLevel LOGLEVEL) LevelInt() int {
	return int(logLevel)
}

const (
	_ LOGLEVEL = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
	OFF
)

const (
	DashSecondFormat  = "2006-01-02 15:04:05"
	SlashSecondFormat = "2006/01/02 15:04:05"

	DashMillisecondFormat  = "2006-01-02 15:04:05.000"
	SlashMillisecondFormat = "2006/01/02 15:04:05.000"
)

var levelStringMap = map[LOGLEVEL]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WANR",
	ERROR: "ERROR",
	FATAL: "FATAL",
	OFF:   "OFF",
}

type loggerMsg struct {
	Itime  string   `json:"create_time"`
	Ilevel LOGLEVEL `json:"level"`
	Body   string   `json:"body"`
	File   string   `json:"file"`
	Line   int      `json:"line"`
}

func formatLoggerMsg(loggerMsg *loggerMsg) string {
	msg := fmt.Sprintf("%s [%5s] [%s:%d] %s", loggerMsg.Itime, loggerMsg.Ilevel.LevelString(), loggerMsg.File, loggerMsg.Line, loggerMsg.Body)
	return msg
}

// 抽象的Logger接口
type AbstractLogger interface {
	ID() string
	Name() string
	Init() error
	Write(loggerMsg *loggerMsg) error
	Flush()
	LoggerConfig
}

// Logger配置信息
type LoggerConfig interface {
	IsJson() bool
	Level() LOGLEVEL
	SetLevel(loglevel LOGLEVEL)
	//TimeFormat string // "ex 2006-01-02 15:04:05.000"
}

// AbstractLogger具体实现，比如fileLogger, consoleLogger
type AdapterLogger struct {
	Id string // every adapter has a unique ID
	AbstractLogger
}

func (adapter *AdapterLogger) ID() string {
	return adapter.Id
}

type adapterLoggerFunc func() AbstractLogger

var adapters = make(map[string]adapterLoggerFunc)

//Register logger adapter
func Register(adapterName string, loggerAdapter adapterLoggerFunc) {

	if _, ok := adapters[adapterName]; ok {
		panic("logger: logger adapter " + adapterName + " already registered!")
	}
	adapters[adapterName] = loggerAdapter
}

//
type Logger struct {
	globalTimeFormat string           // global timeFormat
	callerFlag       bool             // if set true, use runtime.Caller(), performance will be affected.
	lock             sync.Mutex       //sync lock
	adapterArr       []AbstractLogger // adapter arrays
	msgChan          chan *loggerMsg  // message channel
	isSync           bool             // is sync
	wait             sync.WaitGroup   // process wait
	signalChan       chan string
}

// set all adapter LogLevel
func (logger *Logger) SetGlobalLevel(loglevel LOGLEVEL) {
	for _, adapter := range logger.adapterArr {
		adapter.SetLevel(loglevel)
	}
}

func (logger *Logger) ShowLevel() map[string]string {
	res := make(map[string]string, len(logger.adapterArr))
	for _, adapter := range logger.adapterArr {
		res[adapter.ID()] = adapter.Level().LevelString()
	}
	return res
}

//start attach a logger adapter after lock
//return : error
func (logger *Logger) Attach(adapter AbstractLogger) error {
	logger.lock.Lock()
	defer logger.lock.Unlock()

	return logger.attach(adapter)
}

//attach a logger adapter
//return : error
func (logger *Logger) attach(adapter AbstractLogger) error {
	for _, v := range logger.adapterArr {
		if v.ID() == adapter.ID() {
			printError("logger: adapter [%s] already attached!", adapter.ID())
		}
	}
	logFun, ok := adapters[adapter.Name()]
	if !ok {
		printError("logger: adapter %s is not registered", adapter.Name())
	}

	adapterLog := logFun()

	if err := adapterLog.Init(); err != nil {
		printError("logger: adapter %s init failed, error: %s", adapter.ID(), err.Error())
	}

	logger.adapterArr = append(logger.adapterArr, adapter)

	return nil
}

//start detach a logger adapter after lock
//return : error
func (logger *Logger) Detach(adapterID string) error {
	logger.lock.Lock()
	defer logger.lock.Unlock()

	return logger.detach(adapterID)
}

//detach a logger adapter
//return : error
func (logger *Logger) detach(adapterID string) error {
	for i, v := range logger.adapterArr {
		if v.ID() == adapterID {
			logger.adapterArr = append(logger.adapterArr[:i], logger.adapterArr[i+1:]...)
			break
		}
	}
	return nil
}

//set logger synchronous false
//params : buf , if not set, default cap(msgChan) = 100, if set, cap(msgChan) = buf[0]
func (logger *Logger) SetAsync(buf ...int) {
	logger.lock.Lock()
	defer logger.lock.Unlock()
	logger.isSync = false

	msgChanLen := 100
	if len(buf) > 0 {
		msgChanLen = buf[0]
	}

	logger.msgChan = make(chan *loggerMsg, msgChanLen)
	logger.signalChan = make(chan string, 1)

	if !logger.isSync {
		go func() {
			defer func() {
				e := recover()
				if e != nil {
					fmt.Printf("%v", e)
				}
			}()
			logger.startAsyncWrite()
		}()
	}
}

//writers log message
//return : error
func (logger *Logger) logInternalWithCaller(level LOGLEVEL, timeFormat string, msg string, withCaller bool) error {

	file := "null"
	line := 0

	if withCaller {
		_, file, line, _ = runtime.Caller(2)
	}
	_, filename := path.Split(file)

	loggerMsg := &loggerMsg{
		Itime:  time.Now().Format(timeFormat),
		Ilevel: level,
		Body:   msg,
		File:   filename,
		Line:   line,
	}

	if !logger.isSync {
		logger.wait.Add(1)
		logger.msgChan <- loggerMsg
	} else {
		logger.writeToOutputs(loggerMsg)
	}

	return nil
}

func (logger *Logger) SetGlobalTimeFormat(timeFormat string) {
	logger.globalTimeFormat = timeFormat
}

func (logger *Logger) logInternal(level LOGLEVEL, msg string) {
	logger.logInternalWithCaller(level, logger.globalTimeFormat, msg, logger.callerFlag)
}

//sync writers message to loggerOutputs
//params : loggerMsg
func (logger *Logger) writeToOutputs(loggerMsg *loggerMsg) {
	for _, adapter := range logger.adapterArr {
		// writers level

		err := adapter.Write(loggerMsg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "logger: unable writers loggerMsg to adapter:%v, error: %v\n", adapter.ID(), err)
		}
	}
}

//start async writers by read logger.msgChan
func (logger *Logger) startAsyncWrite() {
	for {
		select {
		case loggerMsg := <-logger.msgChan:
			logger.writeToOutputs(loggerMsg)
			logger.wait.Done()
		case signal := <-logger.signalChan:
			if signal == "flush" {
				logger.flush()
			}
		}
	}
}

//flush msgChan data
func (logger *Logger) flush() {
	if !logger.isSync {
		for {
			if len(logger.msgChan) > 0 {
				loggerMsg := <-logger.msgChan
				logger.writeToOutputs(loggerMsg)
				logger.wait.Done()
				continue
			}
			break
		}
		for _, adapter := range logger.adapterArr {
			adapter.Flush()
		}
	}
}

//if SetAsync() or logger.isSync() is false, must call Flush() to flush msgChan data
func (logger *Logger) Flush() {
	if !logger.isSync {
		logger.signalChan <- "flush"
		logger.wait.Wait()
		return
	}
	logger.flush()
}

func (logger *Logger) Fatal(msg string) {
	logger.logInternal(FATAL, msg)
}

func (logger *Logger) Fatalf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	logger.logInternal(FATAL, msg)
}

func (logger *Logger) Error(msg string) {
	logger.logInternal(ERROR, msg)
}

func (logger *Logger) Errorf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	logger.logInternal(ERROR, msg)
}

func (logger *Logger) Warn(msg string) {
	logger.logInternal(WARN, msg)
}

func (logger *Logger) Warnf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	logger.logInternal(WARN, msg)
}

func (logger *Logger) Info(msg string) {
	logger.logInternal(INFO, msg)
}

func (logger *Logger) Infof(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	logger.logInternal(INFO, msg)
}

func (logger *Logger) Debug(msg string) {
	logger.logInternal(DEBUG, msg)
}

func (logger *Logger) Debugf(format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	logger.logInternal(DEBUG, msg)
}

func printError(format string, v ...interface{}) {
	fmt.Printf(format, v...)
	os.Exit(0)
}

//get default logger
//return logger
func GetLogger() *Logger {
	//default adapter console
	consoleAdapter := NewConsoleAdapter(INFO, true, false)
	return NewLogger(DashMillisecondFormat, true, consoleAdapter)

}

// new logger
func NewLogger(globalTimeFormat string, callerFlag bool, loggerAdapters ...AbstractLogger) *Logger {
	logger := &Logger{
		globalTimeFormat: globalTimeFormat,
		callerFlag:       callerFlag,
		adapterArr:       []AbstractLogger{},
		msgChan:          make(chan *loggerMsg, 10),
		isSync:           true,
		wait:             sync.WaitGroup{},
		signalChan:       make(chan string, 1),
	}
	for _, adapter := range loggerAdapters {
		logger.attach(adapter)
	}
	return logger
}


