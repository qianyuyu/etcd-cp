package logutil

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var DefaultZapLoggerConfig = zap.Config{
	Level: zap.NewAtomicLevelAt(ConverToZapLevel(DefalutLogLevel)),

	Development: false,
	Sampling: &zap.SamplingConfig{
		Initial:    100,
		Thereafter: 100,
	},

	Encoding: "json",

	EncoderConfig: zapcore.EncoderConfig{
		TimeKey:        "ts",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	},

	//如果需要丢弃信息 则使用/dev/null
	OutputPaths:      []string{"stderr"},
	ErrorOutputPaths: []string{"stderr"},
}

func MergeOutputPaths(cfg zap.Config) zap.Config {
	outputs := make(map[string]struct{})
	outputPaths := make([]string, 0)

	for _, v := range cfg.OutputPaths {
		outputs[v] = struct{}{}
	}
	if _, ok := outputs["/dev/null"]; ok {
		outputPaths = []string{"dev/null"}
	} else {
		for k := range outputs {
			outputPaths = append(outputPaths, k)
		}
	}
	cfg.OutputPaths = outputPaths

	errOutputs := make(map[string]struct{})
	errOutputPaths := make([]string, 0)

	for _, v := range cfg.ErrorOutputPaths {
		errOutputs[v] = struct{}{}
	}
	if _, ok := errOutputs["/dev/null"]; ok {
		errOutputPaths = []string{"dev/null"}
	} else {
		for k := range outputs {
			errOutputPaths = append(outputPaths, k)
		}
	}
	cfg.ErrorOutputPaths = errOutputPaths

	return cfg
}
