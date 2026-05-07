package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/normahq/codex-acp-bridge/cmd/codex-acp-bridge/cmd"
)

func main() {
	os.Exit(run())
}

func run() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := command.Command().ExecuteContext(ctx); err != nil {
		return 1
	}
	return 0
}
