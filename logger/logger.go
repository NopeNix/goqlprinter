package logger

import (
	"log"
	"os"
	"strings"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARNING
	ERROR
)

var (
	currentLevel LogLevel
	initialized  bool
)

func Init(level string) {
	levelStr := strings.ToUpper(level)

	switch levelStr {
	case "DEBUG":
		currentLevel = DEBUG
		log.SetOutput(os.Stdout)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	case "INFO":
		currentLevel = INFO
		log.SetOutput(os.Stdout)
		log.SetFlags(log.LstdFlags)
	case "WARNING":
		currentLevel = WARNING
		log.SetOutput(os.Stdout)
		log.SetFlags(log.LstdFlags)
	case "ERROR":
		currentLevel = ERROR
		log.SetOutput(os.Stdout)
		log.SetFlags(log.LstdFlags)
	default:
		currentLevel = INFO
		log.SetOutput(os.Stdout)
		log.SetFlags(log.LstdFlags)
	}

	initialized = true
}

func GetLevel() LogLevel {
	return currentLevel
}

func Debug(format string, v ...any) {
	if !initialized {
		Init("")
	}
	if currentLevel <= DEBUG {
		log.Printf("[DEBUG] "+format, v...)
	}
}

func Info(format string, v ...any) {
	if !initialized {
		Init("")
	}
	if currentLevel <= INFO {
		log.Printf("[INFO] "+format, v...)
	}
}

func Warning(format string, v ...any) {
	if !initialized {
		Init("")
	}
	if currentLevel <= WARNING {
		log.Printf("[WARNING] "+format, v...)
	}
}

func Error(format string, v ...any) {
	if !initialized {
		Init("")
	}
	if currentLevel <= ERROR {
		log.Printf("[ERROR] "+format, v...)
	}
}
