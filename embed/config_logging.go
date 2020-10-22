package embed

import (
	"etcd-cp/pkg/logutil"
	"fmt"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/grpclog"
	"sync"
)

var grpcLogOnce = new(sync.Once)

func (cfg *Config) setupLogging() error {
	switch cfg.Logger {
	case "capnslog":
		return fmt.Errorf("--logger=capnslog is removed in v3.5")
	case "zap":
		if len(cfg.LogOutputs) == 0 {
			cfg.LogOutputs = []string{DefaultLogOutput}
		}

		if len(cfg.LogOutputs) > 1 {
			for _, l := range cfg.LogOutputs {
				if l == DefaultLogOutput {
					return fmt.Errorf("multi logoutput for %q is not supported yet", DefaultLogOutput)
				}
			}
		}

		outputPaths, errOutputPaths := make([]string, 0), make([]string, 0)
		isJournal := false

		for _, v := range cfg.LogOutputs {
			//原代码写的那么啰嗦,精简了一下，并不影响阅读
			switch v {
			case DefaultLogOutput:
				outputPaths = append(outputPaths, StdOutLogOutput)
				errOutputPaths = append(errOutputPaths, StdErrLogOutput)
			case JournalLogOutput:
				isJournal = true
			default:
				outputPaths = append(outputPaths, v)
				errOutputPaths = append(errOutputPaths, v)
			}
		}

		if !isJournal {
			copied := logutil.DefaultZapLoggerConfig
			copied.OutputPaths = outputPaths
			copied.ErrorOutputPaths = errOutputPaths
			copied = logutil.MergeOutputPaths(copied)
			copied.Level = zap.NewAtomicLevelAt(logutil.ConverToZapLevel(cfg.LogLevel))

			if cfg.LogLevel == "debug" {
				grpc.EnableTracing = true
			}

			if cfg.ZapLoggerBuilder == nil {
				cfg.ZapLoggerBuilder = func(config *Config) error {
					var err error
					config.logger, err = copied.Build()
					if err != nil {
						return nil
					}

					//TODO 这里为什么会加锁？ 暂时没搞清
					config.loggerMu.Lock()
					defer config.loggerMu.Unlock()
					config.loggerConfig = &copied
					config.loggerCore = nil
					config.loggerWriteSyncer = nil

					//这里开始初始化 grpc服务端的日志器, 说白了让grpc也用zap做日志器
					grpcLogOnce.Do(func() {
						var gl grpclog.LoggerV2
						gl, err = logutil.NewGRPCLoggerV2(copied)
						if err == nil {
							grpclog.SetLoggerV2(gl)
						}
					})
					return nil
				}
			}

		} else {

		}

	default:
		return fmt.Errorf("unknown logger option %q", cfg.Logger)
	}

	return nil
}
