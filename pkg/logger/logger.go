package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var logger *zap.Logger
var once sync.Once

func GetZapLogger() (*zap.Logger, error) {
	var err error
	once.Do(func() {
		// if config.Config.Server.Debug {
		// 	logger, err = zap.NewDevelopment()
		// } else {
		// 	logger, err = zap.NewProduction()
		// }
		config := zap.NewProductionConfig()
		config.DisableCaller = true
		config.DisableStacktrace = true
		config.Level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
		logger, err = config.Build()
	})

	return logger, err
}
