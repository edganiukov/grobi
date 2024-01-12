package main

import (
	"context"
	"log/slog"
	"os/exec"
)

func RunOnFailure(ctx context.Context, commands []string) {
	slog.Error("encountered error, executing on-failure commands")
	if r := recover(); r != nil {
		slog.Error("recovered error", "err", r)
	}

	for _, cmd := range commands {
		slog.Info("running on_failure command", "command", cmd)
		if err := RunCommand(ctx, exec.Command("sh", "-c", cmd)); err != nil {
			slog.Error("on_failure command failed", "err", err)
		}
	}
}
