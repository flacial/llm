package log

import (
	"io"
	"os"
	"path/filepath"

	"github.com/mattn/go-isatty"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

var Logger zerolog.Logger

func InitLggger(verbose, debug bool, logFile string) {
	// Default level is info
	zerolog.SetGlobalLevel(zerolog.InfoLevel)

	consoleWriter := zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: zerolog.TimeFormatUnix,
		// Disable color for non-TTY
		NoColor: !isatty.IsTerminal(os.Stderr.Fd()),
	}

	var fileWriter io.Writer
	if logFile == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			zlog.Err(err).Msg("Failed to get user home directory, unable to set default log file path.")
			fileWriter = io.Discard
		} else {
			logDir := filepath.Join(homeDir, ".llm", "logs")
			if err := os.MkdirAll(logDir, 0755); err != nil {
				zlog.Err(err).Str("path", logDir).Msg("Failed to create log directory.")
				fileWriter = io.Discard
			} else {
				logFile = filepath.Join(logDir, "llm.log")
				logFileHandle, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					zlog.Err(err).Str("path", logFile).Msg("Failed to open log file.")
					fileWriter = io.Discard
				} else {
					fileWriter = logFileHandle
				}
			}
		}
	} else {
		logFileHandle, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			zlog.Err(err).Str("path", logFile).Msg("Failed to open specified log file.")
			fileWriter = io.Discard
		} else {
			fileWriter = logFileHandle
		}
	}

	var writers io.Writer
	if fileWriter != nil {
		writers = io.MultiWriter(consoleWriter, fileWriter)
	} else {
		writers = consoleWriter
	}

	Logger = zerolog.New(writers).With().Timestamp().Logger()

	if debug {
		Logger = Logger.Level(zerolog.DebugLevel)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	} else if verbose {
		Logger = Logger.Level(zerolog.InfoLevel)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	} else {
		Logger = Logger.Level(zerolog.WarnLevel)
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	zlog.Logger = Logger

	Logger.Debug().Str("log_file_path", logFile).Msg("Logger initialized.")
}
