package logger

import (
	"context"
	"os"
	"sync"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/instill-ai/model-backend/config"
)

var once sync.Once
var core zapcore.Core

// GetZapLogger returns an instance of zap logger
func GetZapLogger(ctx context.Context) (*zap.Logger, error) {
	var err error
	once.Do(func() {
		// debug and info level enabler
		debugInfoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.DebugLevel || level == zapcore.InfoLevel
		})

		// info level enabler
		infoLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.InfoLevel
		})

		// warn, error and fatal level enabler
		warnErrorFatalLevel := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return level == zapcore.WarnLevel || level == zapcore.ErrorLevel || level == zapcore.FatalLevel
		})

		// write syncers
		stdoutSyncer := zapcore.Lock(os.Stdout)
		stderrSyncer := zapcore.Lock(os.Stderr)

		// tee core
		if config.Config.Server.Debug {
			core = zapcore.NewTee(
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
					stdoutSyncer,
					debugInfoLevel,
				),
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
					stderrSyncer,
					warnErrorFatalLevel,
				),
			)
		} else {
			core = zapcore.NewTee(
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
					stdoutSyncer,
					infoLevel,
				),
				zapcore.NewCore(
					zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
					stderrSyncer,
					warnErrorFatalLevel,
				),
			)
		}
	})
	// finally construct the logger with the tee core
	// and add hooks to inject logs to traces
	logger := zap.New(core).WithOptions(
		zap.Hooks(func(entry zapcore.Entry) error {
			span := trace.SpanFromContext(ctx)
			if !span.IsRecording() {
				return nil
			}

			span.AddEvent("log", trace.WithAttributes(
				attribute.KeyValue{
					Key:   "log.severity",
					Value: attribute.StringValue(entry.Level.String()),
				},
				attribute.KeyValue{
					Key:   "log.message",
					Value: attribute.StringValue(entry.Message),
				},
			))
			if entry.Level >= zap.ErrorLevel {
				span.SetStatus(codes.Error, entry.Message)
			} else {
				span.SetStatus(codes.Ok, "")
			}

			return nil
		}))

	return logger, err
}
