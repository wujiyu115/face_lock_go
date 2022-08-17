package main

import (
	"fmt"
	"path"
	"runtime"
	"strings"
	"time"

	formatter "github.com/antonfisher/nested-logrus-formatter"
	rotatelogs "github.com/lestrrat-go/file-rotatelogs"
	"github.com/rifflock/lfshook"
	log "github.com/sirupsen/logrus"
)

func logInit(cfg *Cfg) {
	log.SetReportCaller(true)
	logFormatter := &formatter.Formatter{
		NoColors:        true,
		TimestampFormat: "2006-01-02 15:03:04",
		CallerFirst:     false,
		CustomCallerFormatter: func(f *runtime.Frame) string {
			s := strings.Split(f.Function, ".")
			funcName := s[len(s)-1]
			return fmt.Sprintf(" [%s:%d][%s()]", path.Base(f.File), f.Line, funcName)
		},
	}
	log.SetFormatter(logFormatter)
	/*
		file, err := os.OpenFile(cfg.LogFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		writers := []io.Writer{
			file,
			os.Stdout}
		fileAndStdoutWriter := io.MultiWriter(writers...)
		if err == nil {
			log.SetOutput(fileAndStdoutWriter)
		} else {
			log.Error("failed to log to file.")
		}
	*/

	//set log level
	level, err := log.ParseLevel(cfg.LogLevel)
	checkIfError(err)
	log.SetLevel(level)

	// 设置 rotatelogs
	var logWriter *rotatelogs.RotateLogs
	logWriter, err = rotatelogs.New(
		// file name
		cfg.LogFileName+".%Y%m%d.log",
		// create link point to new log file
		// rotatelogs.WithLinkName(cfg.LogFileName),
		// set max age
		rotatelogs.WithMaxAge(time.Duration(cfg.LogFileMaxAge)*time.Hour*24),
		// set rotation time
		rotatelogs.WithRotationTime(time.Duration(cfg.LogFileRotationTime)*time.Hour),
		// rotatelogs.ForceNewFile(),
	)
	checkIfError(err)

	writeMap := lfshook.WriterMap{
		log.InfoLevel:  logWriter,
		log.FatalLevel: logWriter,
		log.DebugLevel: logWriter,
		log.WarnLevel:  logWriter,
		log.ErrorLevel: logWriter,
		log.PanicLevel: logWriter,
	}

	lfHook := lfshook.NewHook(writeMap, logFormatter)
	log.AddHook(lfHook)
}
