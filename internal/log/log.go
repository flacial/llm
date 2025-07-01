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

	var fileWriter io.Writer = io.Discard // Default to discard
	var finalLogPath string

	if logFile != "" {
		finalLogPath = logFile
	} else {
		// Follow XDG Directory spec of storing non-portable data in state home
		// https://specifications.freedesktop.org/basedir-spec/latest/#variables
		xdgStateHome := os.Getenv("XDG_STATE_HOME")
		if xdgStateHome == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				zlog.Err(err).Msg("Failed to get user home directory, cannot set default log file path.")
			} else {
				xdgStateHome = filepath.Join(home, ".local", "state")
			}
		}

		if xdgStateHome != "" {
			finalLogPath = filepath.Join(xdgStateHome, "llm", "llm.log")
		}
	}

	if finalLogPath != "" {
		if err := os.MkdirAll(filepath.Dir(finalLogPath), 0755); err != nil {
			zlog.Err(err).Str("path", finalLogPath).Msg("Failed to create log directory.")
		} else {
			logFileHandle, err := os.OpenFile(finalLogPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				zlog.Err(err).Str("path", finalLogPath).Msg("Failed to open log file.")
			} else {
				fileWriter = logFileHandle
			}
		}
	}

	writers := io.MultiWriter(consoleWriter, fileWriter)
	Logger = zerolog.New(writers).With().Timestamp().Logger()

	switch {
	case debug:
		Logger = Logger.Level(zerolog.DebugLevel)
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case verbose:
		Logger = Logger.Level(zerolog.InfoLevel)
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	default:
		Logger = Logger.Level(zerolog.WarnLevel)
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	}

	zlog.Logger = Logger

	if finalLogPath != "" {
		Logger.Debug().Str("log_file_path", finalLogPath).Msg("Logger initialized.")
	} else {
		Logger.Debug().Msg("Logger initialized without a log file.")
	}
}
