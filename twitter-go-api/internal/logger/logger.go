package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// InitLogger configures the global zerolog instance.
func InitLogger(env string) {
	zerolog.TimeFieldFormat = time.RFC3339

	if env == "development" {
		// Pretty console output for unhandled development logging
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339})
	} else {
		// Strict JSON for production
		log.Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()
	}
}
