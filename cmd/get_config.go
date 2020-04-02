package cmd

import (
	"blaster/core"
	"time"
)

func GetConfig() *core.Config {
	return &core.Config{
		HandlerCommand:      handlerCommand,
		HandlerArgs:         handlerArgv,
		HandlerURL:          handlerURL,
		RetryCount:          retryCount,
		RetryDelay:          time.Second * time.Duration(retryDelaySeconds),
		StartupDelaySeconds: startupDelaySeconds,
		MaxHandlers:         maxHandlers,
		EnableVersboseLog:   enableVerboseLog,
	}
}
