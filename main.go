package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"time"
)

var (
	opts Options
)

// Options contains all global options.
type Options struct {
	Verbose      bool
	Config       string
	DryRun       bool
	PollInterval time.Duration
	ActivePoll   bool
	Pause        time.Duration
}

func main() {
	flag.BoolVar(&opts.Verbose, "verbose", false, "Enable verbose logging")
	flag.StringVar(&opts.Config, "config", "", "The path to a config file.")
	flag.BoolVar(&opts.DryRun, "dry-run", false, "Enable dry-run mode: print commands instead of running them.")
	flag.DurationVar(&opts.PollInterval, "interval", 2*time.Second, "Duration between polls. Set to zero to disable polling.")
	flag.BoolVar(&opts.ActivePoll, "active-poll", false, "Force xrandr to re-detect outputs during polling.")
	flag.DurationVar(&opts.Pause, "pause", 0, "Number of seconds to pause after a change was executed.")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg, err := readConfig(opts.Config)
	if err != nil {
		slog.Error("could not read config file", "err", err)
		os.Exit(1)
	}

	mm := &MonitorManager{
		Config: cfg,
	}

	if len(os.Args) == 1 {
		fmt.Printf("Please specify a command to run: %s", strings.Join(availableCommands(subCommands), ", "))
		os.Exit(1)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	subCmdName := os.Args[1]
	args := os.Args[2:]

	if subCmdName == "help" {
		helpSubCmd(ctx, args)
		os.Exit(0)
	}

	subCmd, ok := resolveCommand(subCommands, subCmdName)
	if !ok {
		fmt.Printf("Unknown command %q.\nPlease specify one of the commands: %s\n", subCmdName, strings.Join(availableCommands(subCommands), ", "))
		os.Exit(1)
	}

	if err := subCmd.Run(ctx, mm, args); err != nil {
		slog.Error("command failed", "command", subCmdName, "err", err)
		os.Exit(1)
	}

	os.Exit(0)
}
