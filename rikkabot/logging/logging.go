// @Author Clover
// @Data 2024/7/18 上午10:24:00
// @Desc 日志输出
package logging

import (
	"flag"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"sync"
)

var (
	logfile *os.File
	once    sync.Once
)

func init() {
	debugflag := flag.Bool("debug", false, "debug mode")
	flag.Parse()

	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stderr})
	logfile, err := os.OpenFile("rikka.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Error().Msgf("Error opening file: %v", err)
	}
	multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stderr}, logfile)

	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
	if *debugflag {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else {
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}

// 关闭日志文件
func Close() {
	once.Do(func() {
		if logfile != nil {
			err := logfile.Close()
			if err != nil {
				log.Error().Msgf("Error closing log file: %v", err)
			}
			logfile = nil
		}
	})
}

// 定义简化的日志函数
func Info(msg string, fields ...map[string]interface{}) {
	event := log.Info()
	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

func Error(msg string, fields ...map[string]interface{}) {
	event := log.Error()
	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

func ErrorWithErr(err error, msg string, fields ...map[string]interface{}) {
	event := log.Error()
	event.Err(err)
	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

func Debug(msg string, fields ...map[string]interface{}) {
	event := log.Debug()
	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

func Warn(msg string, fields ...map[string]interface{}) {
	event := log.Warn()
	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
}

func Fatal(msg string, exitCode int, fields ...map[string]interface{}) {
	event := log.Fatal()
	for _, field := range fields {
		for k, v := range field {
			event = event.Interface(k, v)
		}
	}
	event.Msg(msg)
	os.Exit(exitCode)
}
