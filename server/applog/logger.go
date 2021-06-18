package applog

import (
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/st-user/ojm-drone-local/appos"
	"github.com/st-user/ojm-drone-local/env"
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
}

func NewLogger() Logger {
	levelInt := 1
	levelStr := strings.ToUpper(env.Get("LOG_LEVEL"))
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

	deleteFiles()
	prepare()
	writer := createWriter()

	return Logger{
		level: levelInt,
		debug: log.New(writer, "[DEBUG]: ", log.Ldate|log.Ltime),
		info:  log.New(writer, "[INFO]: ", log.Ldate|log.Ltime),
		warn:  log.New(writer, "[WARN]: ", log.Ldate|log.Ltime),
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

func createWriter() io.Writer {
	logFilePath := createFilepath()
	fileToDump, err := os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	if !env.GetBool("LOG_OUTPUT_CONSOLE") {
		return fileToDump
	}
	return io.MultiWriter(fileToDump, os.Stdout)
}

func prepare() {
	dir := createOutputDirPath()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.Mkdir(dir, 0744); err != nil {
			panic(err)
		}
	}
}

func deleteFiles() {
	basepath := createOutputDirPath()
	files, err := ioutil.ReadDir(basepath)
	if err != nil {
		log.Fatal(err)
	}
	fileBaseName := env.Get("LOG_FILE_BASE_NAME")
	daysToReserve := env.GetInt("LOG_DAYS_TO_RESERVER")
	deleteAfter := time.Now().AddDate(0, 0, -daysToReserve)

	for _, f := range files {

		filename := f.Name()
		if strings.HasPrefix(filename, fileBaseName+"-") {

			datePartsStr := strings.Replace(filename, fileBaseName+"-", "", 1)
			datePartsStr = strings.Replace(datePartsStr, ".log", "", 1)
			dateParts := strings.Split(datePartsStr, "-")

			year, err := strconv.Atoi(dateParts[0])
			if err != nil {
				fmt.Printf("Illigal filename %v", filename)
				fmt.Println()
				continue
			}

			month, err := strconv.Atoi(dateParts[1])
			if err != nil {
				fmt.Printf("Illigal filename %v", filename)
				fmt.Println()
				continue
			}

			day, err := strconv.Atoi(dateParts[2])
			if err != nil {
				fmt.Printf("Illigal filename %v", filename)
				fmt.Println()
				continue
			}

			fileDate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Now().Location())

			if deleteAfter.After(fileDate) {
				err = os.Remove(filepath.Join(basepath, filename))
				if err != nil {
					fmt.Printf("Failed to remove %v", filename)
					fmt.Println()
				} else {
					fmt.Printf("Removes %v", filename)
					fmt.Println()
				}
			}
		}

	}
}

func createFilepath() string {
	now := time.Now()
	filename := createFilename(now.Year(), int(now.Month()), now.Day())
	return filepath.Join(createOutputDirPath(), filename)
}

func createOutputDirPath() string {
	return filepath.Join(getDir(), env.Get("LOG_OUTPUT_DIR"))
}

func createFilename(year int, month int, day int) string {
	fileBaseName := env.Get("LOG_FILE_BASE_NAME")
	return fmt.Sprintf(fileBaseName+"-%v-%v-%v.log", year, month, day)
}

func getDir() string {
	dir := os.Getenv("LOG_BASE_DIR")
	if len(dir) > 0 {
		return dir
	}
	return appos.BaseDir()
}
