package main

import (
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	debug = 0
	info  = 1
	warn  = 2
)

type Logger struct {
	level int
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	dump  *log.Logger
}

func NewLogger(levelStr string) Logger {
	levelInt := 1
	levelStr = strings.ToUpper(levelStr)
	switch levelStr {
	case "DEBUG":
		levelInt = debug
	case "INFO":
		levelInt = info
	case "WARN":
		levelInt = warn
	default:
		log.Fatalf("Invalid log level: %v", levelStr)
	}

	dumpFilePath := filepath.Join(BaseDir(), "dump", "server.log")
	_dumpFilePath := os.Getenv("DUMP_LOG_FILE_PATH")
	if len(_dumpFilePath) > 0 {
		dumpFilePath = _dumpFilePath
	}
	fileToDump, err := os.OpenFile(dumpFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Fatal(err)
	}

	return Logger{
		level: levelInt,
		debug: log.New(os.Stdout, "[DEBUG]: ", log.Ldate|log.Ltime),
		info:  log.New(os.Stdout, "[INFO]: ", log.Ldate|log.Ltime),
		warn:  log.New(os.Stdout, "[WARN]: ", log.Ldate|log.Ltime),
		dump:  log.New(fileToDump, "", log.Ltime),
	}
}

func (logger *Logger) Debug(format string, v ...interface{}) {
	if logger.level <= debug {
		logger.debug.Printf(format, v...)
	}
}

func (logger *Logger) Info(format string, v ...interface{}) {
	if logger.level <= info {
		logger.info.Printf(format, v...)
	}
}

func (logger *Logger) Warn(format string, v ...interface{}) {
	if logger.level <= warn {
		logger.warn.Printf(format, v...)
	}
}

func (logger *Logger) Dump(format string, v ...interface{}) {
	logger.dump.Printf(format, v...)
}
