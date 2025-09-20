// Ref: GPT5 - OpenAI
// io.Writer wrapper for the Bubbletea UILogger
package main

import (
	"bytes"
	"strings"
)

// UILoggerWriter implements io.Writer so it can be used in libraries
// expecting an Output writer. Each Write call emits a log line
// into the UILogger as INFO by default.
type UILoggerWriter struct {
	logger *UILogger
	level  logLevel
	buf    bytes.Buffer
}

// NewUILoggerWriter creates a writer that forwards to UILogger.Info by default.
func NewUILoggerWriter(l *UILogger) *UILoggerWriter {
	return &UILoggerWriter{
		logger: l,
		level:  levelInfo,
	}
}

// SetLevel changes the log level for emitted lines.
func (w *UILoggerWriter) SetLevel(level logLevel) {
	w.level = level
}

// Write implements io.Writer. It buffers until newline, then emits whole lines.
func (w *UILoggerWriter) Write(p []byte) (n int, err error) {
	n = len(p)
	for _, b := range p {
		if b == '\n' {
			line := strings.TrimRight(w.buf.String(), "\r\n")
			w.emit(line)
			w.buf.Reset()
		} else {
			_ = w.buf.WriteByte(b)
		}
	}
	return n, nil
}

func (w *UILoggerWriter) emit(line string) {
	if line == "" {
		return
	}
	switch w.level {
	case levelWarn:
		w.logger.Warn(line)
	case levelError:
		w.logger.Error(line)
	case levelSuccess:
		w.logger.Success(line)
	default:
		w.logger.Info(line)
	}
}
