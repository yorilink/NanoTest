package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

type RunLogger struct {
	file *os.File
	info *log.Logger
	err  *log.Logger
	path string
}

func NewRunLogger(dir string) (*RunLogger, error) {
	if dir == "" {
		dir = filepath.Join("robot", "logs")
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	name := fmt.Sprintf("robot-%s-%d.log", time.Now().Format("20060102-150405"), os.Getpid())
	path := filepath.Join(dir, name)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	return &RunLogger{
		file: file,
		info: log.New(io.MultiWriter(os.Stdout, file), "[INFO] ", log.LstdFlags|log.Lmicroseconds),
		err:  log.New(io.MultiWriter(os.Stderr, file), "[ERROR] ", log.LstdFlags|log.Lmicroseconds),
		path: path,
	}, nil
}

func (l *RunLogger) Path() string {
	if l == nil {
		return ""
	}
	return l.path
}

func (l *RunLogger) Infof(format string, args ...interface{}) {
	if l == nil {
		return
	}
	l.info.Printf(format, args...)
}

func (l *RunLogger) Errorf(format string, args ...interface{}) {
	if l == nil {
		return
	}
	l.err.Printf(format, args...)
}

func (l *RunLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}
