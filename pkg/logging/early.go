package logging

import (
	"fmt"
	"os"
)

type EarlyLog struct{}

func NewEarlyLog() *EarlyLog {
	return &EarlyLog{}
}

func (l *EarlyLog) Error(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "ERROR: "+msg+"\n", args...)
	os.Exit(1)
}

func (l *EarlyLog) Fatal(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "FATAL: "+msg+"\n", args...)
	os.Exit(1)
}

func (l *EarlyLog) Warn(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "WARN: "+msg+"\n", args...)
}

func (l *EarlyLog) Info(msg string, args ...interface{}) {
	fmt.Fprintf(os.Stdout, "INFO: "+msg+"\n", args...)
}
