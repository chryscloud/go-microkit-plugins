// Copyright 2020 Wearless Tech Inc All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// ZapLogger - logger
type ZapLogger struct {
	Zap       *zap.Logger
	zapConfig zap.Config
}

// NewZapLogger initiates zap logging with JSON encoding output for fluentd logging (stdout, stderr)
func NewZapLogger(logLevel string) (*ZapLogger, error) {
	level := getLogLevel(logLevel)
	zapConf := zapConfig(level)
	zapLog, err := initZap(zapConf)
	if err != nil {
		return nil, err
	}
	zap.RedirectStdLog(zapLog)

	return &ZapLogger{Zap: zapLog, zapConfig: zapConf}, nil
}

// Error outputs errors to fluentd with stacktraces
func (z *ZapLogger) Error(keyvals ...interface{}) {
	z.Zap.Sugar().Error(keyvals)
}

// Warn - outputs warnings to fluentd fluentd enagled logging
func (z *ZapLogger) Warn(keyvals ...interface{}) {
	z.Zap.Sugar().Warn(keyvals)
}

// Info - outputs info messages to fluentd enagled logging
func (z *ZapLogger) Info(keyvals ...interface{}) {
	z.Zap.Sugar().Info(keyvals)
}

func zapConfig(level zap.AtomicLevel) zap.Config {

	zapEncoderConfig := zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	zapConfig := zap.Config{
		Level:       level,
		Development: false,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding:         "json",
		EncoderConfig:    zapEncoderConfig,
		OutputPaths:      []string{"stderr"},
		ErrorOutputPaths: []string{"stderr"},
	}
	return zapConfig
}

func getLogLevel(logLevel string) zap.AtomicLevel {
	level := zap.NewAtomicLevelAt(zapcore.InfoLevel)
	switch logLevel {
	case "debug":
		level = zap.NewAtomicLevelAt(zapcore.DebugLevel)
	case "info":
		level = zap.NewAtomicLevelAt(zapcore.InfoLevel)
	case "warn":
		level = zap.NewAtomicLevelAt(zapcore.WarnLevel)
	case "error":
		level = zap.NewAtomicLevelAt(zapcore.ErrorLevel)
	case "fatal":
		level = zap.NewAtomicLevelAt(zapcore.FatalLevel)
	case "panic":
		level = zap.NewAtomicLevelAt(zapcore.PanicLevel)
	}
	return level
}

func initZap(zapConfig zap.Config) (*zap.Logger, error) {
	return zapConfig.Build()
}
