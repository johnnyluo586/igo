package log

import "fmt"

var _logger *SimpleLogger

func init() {

	//Register("conn", NewConn)
	Register("console", NewConsole)
	Register("file", NewFileWriter)
	Register("nsq", NewNsqWriter)

	//console logger
	_logger = NewLogger(10000)
	SetLogFuncCall(true)
	SetLogger("console", "")
	// execName := strings.Split(filepath.Base(os.Args[0]), ".")[0]
	// SetLogger("nsq", fmt.Sprintf(`{"prefix":"%v","addr":"http://127.0.0.1:4151/mpub?topic=LOG&binary=true"}`, execName))
}

func Init(filename string, level int) {
	//set logger
	SetLogger("file", fmt.Sprintf(`{"filename":"%v", "level":%v}`, filename, level))
}

// SetLogLevel sets the global log level used by the simple logger.
func SetLevel(l int) {
	_logger.SetLevel(l)
}

func SetLogFuncCall(b bool) {
	_logger.EnableFuncCallDepth(b)
	_logger.SetLogFuncCallDepth(3)
}

// SetLogger sets a new logger.
func SetLogger(adaptername string, config string) error {
	err := _logger.SetLogger(adaptername, config)
	if err != nil {
		return err
	}
	return nil
}

func Emergency(v ...interface{}) {
	_logger.Emergency(fmt.Sprint(v...))
}

func Alert(v ...interface{}) {
	_logger.Alert(fmt.Sprint(v...))
}

// Critical logs a message at critical level.
func Critical(v ...interface{}) {
	_logger.Critical(fmt.Sprint(v...))
}

// Error logs a message at error level.
func Error(v ...interface{}) {
	_logger.Error(fmt.Sprint(v...))
}

// Warning logs a message at warning level.
func Warning(v ...interface{}) {
	_logger.Warning(fmt.Sprint(v...))
}

// compatibility alias for Warning()
func Warn(v ...interface{}) {
	_logger.Warn(fmt.Sprint(v...))
}

func Notice(v ...interface{}) {
	_logger.Notice(fmt.Sprint(v...))
}

// Info logs a message at info level.
func Informational(v ...interface{}) {
	_logger.Informational(fmt.Sprint(v...))
}

// compatibility alias for Warning()
func Info(v ...interface{}) {
	_logger.Info(fmt.Sprint(v...))
}

// Debug logs a message at debug level.
func Debug(v ...interface{}) {
	_logger.Debug(fmt.Sprint(v...))
}

// Trace logs a message at trace level.
// compatibility alias for Warning()
func Trace(v ...interface{}) {
	_logger.Trace(fmt.Sprint(v...))
}

func Emergencyf(format string, v ...interface{}) {
	_logger.Emergency(format, v...)
}

func Alertf(format string, v ...interface{}) {
	_logger.Alert(format, v...)
}

// Critical logs a message at critical level.
func Criticalf(format string, v ...interface{}) {
	_logger.Critical(format, v...)
}

// Error logs a message at error level.
func Errorf(format string, v ...interface{}) {
	_logger.Error(format, v...)
}

// Warning logs a message at warning level.
func Warningf(format string, v ...interface{}) {
	_logger.Warning(format, v...)
}

// compatibility alias for Warning()
func Warnf(format string, v ...interface{}) {
	_logger.Warn(format, v...)
}

func Noticef(format string, v ...interface{}) {
	_logger.Notice(format, v...)
}

// Info logs a message at info level.
func Informationalf(format string, v ...interface{}) {
	_logger.Informational(format, v...)
}

// compatibility alias for Warning()
func Infof(format string, v ...interface{}) {
	_logger.Info(format, v...)
}

// Debug logs a message at debug level.
func Debugf(format string, v ...interface{}) {
	_logger.Debug(format, v...)
}

// Trace logs a message at trace level.
// compatibility alias for Warning()
func Tracef(format string, v ...interface{}) {
	_logger.Trace(format, v...)
}
