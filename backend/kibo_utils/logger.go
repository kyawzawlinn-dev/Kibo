package kibo_utils

import (
	"log"
)

// Format [ Type ] [ Location ] Message

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Cyan   = "\033[36m"
)

var Enabled = true // <-- Change this one key to enable/disable all logs

func Info(msg string, v ...any) {
	if Enabled {
		log.Printf(Green+"[ℹ️INFO]\t"+msg+Reset, v...)
	}
}

func Debug(msg string, v ...any) {
	if Enabled {
		log.Printf(Cyan+"[🪲DEBUG]\t"+msg+Reset, v...)
	}
}

func Warn(msg string, v ...any) {
	log.Printf(Yellow+"[⚠️WARN]\t"+msg+Reset, v...)
}

func Error(msg string, v ...any) {
	log.Printf(Red+"[❗️ERROR]\t"+msg+Reset, v...)
}
