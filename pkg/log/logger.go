// Package log provides a logger interface for logger libraries
// so that dex does not depend on any of them directly.
// It also includes a default implementation using Logrus (used by dex previously).
package log

// Logger serves as an adapter interface for logger libraries
// so that dex does not depend on any of them directly.
type Logger interface {
	WithField(key string, value interface{}) Logger

	Info(msg string)
	Warn(msg string)

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Errorf(format string, args ...interface{})
}