package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"runtime"
)

var (
	version = "dev"

	subCommands = []SubCommand{
		{Name: "version", Desc: "Display the version.", Run: versionSubCmd},
		{Name: "apply", Desc: "Apply a rule to configure the outputs accordingly.", Run: applySubCmd},
		{Name: "rules", Desc: "List the configured rules.", Run: rulesSubCmd},
		{Name: "show", Desc: "Show monitors and IDs.", Run: showSubCmd},
		{Name: "update", Desc: "Update outputs config.", Run: updateSubCmd},
		{Name: "watch", Desc: "Watch for XRANDR changes.", Run: watchSubCmd},
	}
)

type SubCommand struct {
	Name string
	Desc string
	Run  func(context.Context, *MonitorManager, []string) error
}

func resolveCommand(cmds []SubCommand, cmdName string) (SubCommand, bool) {
	for _, c := range cmds {
		if c.Name == cmdName {
			return c, true
		}
	}

	return SubCommand{}, false
}

func availableCommands(cmds []SubCommand) []string {
	cmdNames := make([]string, 0, len(cmds))
	for _, c := range cmds {
		cmdNames = append(cmdNames, c.Name)
	}

	return cmdNames
}

func applySubCmd(ctx context.Context, mm *MonitorManager, args []string) error {
	if err := mm.Apply(ctx, args); err != nil {
		RunOnFailure(ctx, mm.Config.OnFailure)
		return err
	}

	return nil
}

func rulesSubCmd(ctx context.Context, mm *MonitorManager, args []string) error {
	return mm.Rules(ctx, args)
}

func showSubCmd(ctx context.Context, mm *MonitorManager, args []string) error {
	return mm.Show(ctx, args)
}

func updateSubCmd(ctx context.Context, mm *MonitorManager, args []string) error {
	if err := mm.Update(ctx, args); err != nil {
		RunOnFailure(ctx, mm.Config.OnFailure)
		return err
	}

	return nil
}

func watchSubCmd(ctx context.Context, mm *MonitorManager, args []string) error {
	if err := mm.Watch(ctx, args); err != nil {
		RunOnFailure(ctx, mm.Config.OnFailure)
		return err
	}

	return nil
}

func versionSubCmd(ctx context.Context, mm *MonitorManager, args []string) error {
	fmt.Printf(
		"%s version: %s\ncompiled with %s on %s\n",
		os.Args[0], version, runtime.Version(), runtime.GOOS,
	)
	return nil
}

func helpSubCmd(ctx context.Context, args []string) error {
	// program help <subcommand>
	if len(args) > 0 {
		cmdName := args[0]
		cmd, ok := resolveCommand(subCommands, cmdName)
		if !ok {
			fmt.Printf("Unknown sub command %q\n", cmdName)
			return nil
		}
		fmt.Printf("Command description: \n")
		fmt.Printf("\t%s\t%s\n", cmdName, cmd.Desc)
		return nil
	}

	flag.Usage()
	fmt.Printf("\nAvailable commands:\n")
	for _, cmd := range subCommands {
		fmt.Printf("\t%s\t%s\n", cmd.Name, cmd.Desc)
	}

	return nil
}
