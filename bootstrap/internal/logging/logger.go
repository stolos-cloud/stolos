package logging

type Logger interface {
	Debug(s string)
	Info(s string)
	Warn(s string)
	Error(s string)
	Success(s string)
	Debugf(f string, a ...any)
	Infof(f string, a ...any)
	Warnf(f string, a ...any)
	Errorf(f string, a ...any)
	Successf(f string, a ...any)
}
