// Package logging
// @Author Clover
// @Data 2024/7/18 上午10:24:00
// @Desc 日志输出
package logging

import (
	"fmt"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	logfile *os.File
	once    sync.Once
)

const (
	logPath = "./log/rikka.log"
)

func init() {
	zerolog.TimeFieldFormat = "2006-01-02 15:04:05"
	zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stderr})
	_, err := validLogPath(logPath, true)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to open log file")
	}
	logfile, err = os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Error().Msgf("Error opening file: %v", err)
	}
	multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stderr}, logfile)

	log.Logger = zerolog.New(multi).With().Timestamp().Logger()
}

// MonitorLogSize checks the log file size and truncates it if it eceeds the max size.
func MonitorLogSize(maxLogsize int64) {
	for {
		// Get the current log file size
		fi, err := logfile.Stat()
		if err != nil {
			log.Error().Err(err).Msg("Error getting file info")
			continue
		}

		if fi.Size() > maxLogsize {
			log.Info().Msg("Log file size exceeds limit. Clearing log file.")
			clearLogFile()
		}

		// Sleep 4 hour
		time.Sleep(4 * time.Hour)
	}
}

func clearLogFile() {
	var err error
	if err = logfile.Close(); err != nil {
		log.Error().Err(err).Msg("Error closing log file before truncation")
		return
	}

	// Truncate the log file to clear its content
	if err := os.Truncate(logPath, 0); err != nil {
		log.Error().Err(err).Msg("Error truncating log file")
		return
	}

	// Reopen the log file
	logfile, err = os.OpenFile(logPath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatal().Err(err).Msg("Error reopening log file after truncation")
		return
	}

	// Update the zerolog writer the new file descriptor
	multi := zerolog.MultiLevelWriter(zerolog.ConsoleWriter{Out: os.Stderr}, logfile)
	log.Logger = zerolog.New(multi).With().Timestamp().Logger()

	log.Info().Msg("Log file cleared successfully.")
}

// Close 关闭日志文件
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

// Info 定义简化的日志函数
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

func WarnWithErr(err error, msg string, fields ...map[string]interface{}) {
	event := log.Warn()
	event.Err(err)
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

func validLogPath(path string, isCreate bool) (bool, error) {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if isCreate {
			if err := os.MkdirAll(dir, os.ModePerm); err != nil {
				return false, fmt.Errorf("error creating log directory: %v", err)
			}
		}
	}
	return true, nil
}
