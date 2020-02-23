package logger

import (
	"fmt"
	"github.com/sirupsen/logrus"
	"github.com/wuzhc/gopusher/config"
	"path"
	"runtime"
)

var log *logrus.Logger

// FatalLevel,ErrorLevel,WarnLevel,InfoLevel,DebugLevel,TraceLevel
func InitLogger() {
	log = logrus.New()
	log.SetLevel(logrus.Level(config.Cfg.LogLevel))
	log.SetReportCaller(config.Cfg.LogReportCaller)
	log.Formatter = &logrus.TextFormatter{
		CallerPrettyfier: func(f *runtime.Frame) (string, string) {
			filename := path.Base(f.File)
			return fmt.Sprintf("%s()", f.Function), fmt.Sprintf("%s:%d", filename, f.Line)
		},
	}
}

func Log() *logrus.Logger {
	return log
}
