package etcdmain

import (
	"fmt"
	"go.uber.org/zap"
	"os"
)

func startEtcdOrProxyV2() {

	cfg := newConfig()

	err := cfg.parse(os.Args[1:])

	lg, zapError := zap.NewProduction()
	if zapError != nil {
		fmt.Printf("error creating zap logger %v ", zapError)
		os.Exit(1)
	}

	if err != nil {
		lg.Warn("failed to verify flags", zap.Error(err))

		os.Exit(1)
	}

}
