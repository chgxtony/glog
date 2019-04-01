package glog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
)

const CONSOLE_ADAPTER_NAME = "console"

type COLOR []byte

func (color COLOR) String() string {
	return string(color)
}

// console color
var (
	greenBg   = []byte{27, 91, 57, 55, 59, 52, 50, 109}
	whiteBg   = []byte{27, 91, 57, 48, 59, 52, 55, 109}
	yellowBg  = []byte{27, 91, 57, 48, 59, 52, 51, 109}
	redBg     = []byte{27, 91, 57, 55, 59, 52, 49, 109}
	blueBg    = []byte{27, 91, 57, 55, 59, 52, 52, 109}
	magentaBg = []byte{27, 91, 57, 55, 59, 52, 53, 109}
	cyanBg    = []byte{27, 91, 57, 55, 59, 52, 54, 109}
	green     = []byte{27, 91, 51, 50, 109}
	white     = []byte{27, 91, 51, 55, 109}
	yellow    = []byte{27, 91, 51, 51, 109}
	red       = []byte{27, 91, 51, 49, 109}
	blue      = []byte{27, 91, 51, 52, 109}
	magenta   = []byte{27, 91, 51, 53, 109}
	cyan      = []byte{27, 91, 51, 54, 109}
	reset     = []byte{27, 91, 48, 109}
)

func TextWithColor(color COLOR, msg string) string {
	buf := &bytes.Buffer{}
	buf.Write([]byte(color))
	buf.Write([]byte(msg))
	buf.Write([]byte(reset))
	return buf.String()
}

var LevelColorMap = map[LOGLEVEL]COLOR{
	FATAL: magenta,
	ERROR: red,
	WARN:  yellow,
	INFO:  green,
	DEBUG: white,
}

func levelColor(logLevel LOGLEVEL) COLOR {
	return LevelColorMap[logLevel]
}

type ConsoleConfig struct {
	JsonFlag  bool
	LogLevel  LOGLEVEL
	ColorFlag bool //console adapter 独有的配置项
	LoggerConfig
}

func (config *ConsoleConfig) Level() LOGLEVEL {
	return config.LogLevel
}

func (config *ConsoleConfig) SetLevel(loglevel LOGLEVEL) {
	config.LogLevel = loglevel
}

func (config *ConsoleConfig) IsJson() bool {
	return config.JsonFlag
}

func (config *ConsoleConfig) IsColor() bool {
	return config.ColorFlag
}

// adapter console
type ConsoleAdapter struct {
	logger *log.Logger
	ConsoleConfig
	AdapterLogger
}

func (*ConsoleAdapter) Name() string {
	return CONSOLE_ADAPTER_NAME
}

func (adapterConsole *ConsoleAdapter) Init() error {
	fmt.Printf("[GLOG] > [%s adapter] init success\n", adapterConsole.Name())
	return nil
}

func (adapterConsole *ConsoleAdapter) Write(loggerMsg *loggerMsg) error {

	msg := ""
	if adapterConsole.IsJson() {
		jsonByte, _ := json.Marshal(loggerMsg)
		msg = string(jsonByte)
	} else {
		msg = formatLoggerMsg(loggerMsg)
	}

	if adapterConsole.Level() <= loggerMsg.Ilevel {
		if adapterConsole.IsColor() {
			info := TextWithColor(levelColor(loggerMsg.Ilevel), msg)
			return adapterConsole.logger.Output(2, info)
		} else {
			return adapterConsole.logger.Output(2, msg)
		}
	}

	return nil

}

// interface wrapper function, 返回类型必须是接口类型
func NewConsoleAdapter(loglevel LOGLEVEL, color bool, json bool) AbstractLogger {
	consoleConfig := ConsoleConfig{
		JsonFlag:  json,
		ColorFlag: color,
		LogLevel:  loglevel,
	}

	return &ConsoleAdapter{
		logger: log.New(os.Stdout, "", 0),
		ConsoleConfig: consoleConfig,
		AdapterLogger: AdapterLogger{
			Id: "defaultConsole",
		},
	}
}

func init() {
	console := func() AbstractLogger {
		return &ConsoleAdapter{}
		//return NewConsoleAdapter(INFO, true, false)
	}
	Register(CONSOLE_ADAPTER_NAME, console)
}
