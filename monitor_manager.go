package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BurntSushi/xgb"
	"github.com/BurntSushi/xgb/randr"
	"github.com/BurntSushi/xgb/xproto"
)

const eventSendTimeout = 500 * time.Millisecond

type Event struct {
	Event xgb.Event
	Error error
}

type MonitorManager struct {
	Config Config
}

func (m MonitorManager) Apply(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return errors.New("need exactly one rule name as the parameter")
	}

	outputs, err := DetectOutputs()
	if err != nil {
		return err
	}

	ruleName := strings.ToLower(args[0])
	for _, rule := range m.Config.Rules {
		if strings.ToLower(rule.Name) == ruleName {
			slog.Info("found matching rule", "name", rule.Name)
			return ApplyRule(ctx, outputs, rule, m.Config.ExecuteAfter)
		}
	}

	return fmt.Errorf("rule %q not found", ruleName)
}

func (m MonitorManager) Rules(ctx context.Context, args []string) error {
	for _, rule := range m.Config.Rules {
		fmt.Printf("Rule %s: \n", rule.Name)
		fmt.Printf("\tConnected: %s\n", rule.OutputsConnected)
		fmt.Printf("\tDisconnected: %s\n", rule.OutputsDisconnected)
		fmt.Printf("\tPresent: %s\n", rule.OutputsPresent)
		fmt.Printf("\tAbsent: %s\n", rule.OutputsAbsent)
		fmt.Printf("\tConfigureRow: [%s]\n", rule.ConfigureRow)
		fmt.Printf("\tConfigureColumn: [%s]\n", rule.ConfigureColumn)
		fmt.Printf("\tgConfigureSingle: [%s]\n", []*OutputConfig{rule.ConfigureSingle})
		fmt.Printf("\tConfigureCommand: %s\n", rule.ConfigureCommand)
		fmt.Printf("\tExecuteAfter: %s\n", rule.ExecuteAfter)
	}
	return nil
}

func (m MonitorManager) Show(ctx context.Context, args []string) error {
	outputs, err := DetectOutputs()
	if err != nil {
		return err
	}
	for _, output := range outputs {
		if output.Connected {
			fmt.Printf("%- 10s %s\n", output.Name, output.MonitorID)
		}
	}
	return nil
}

func (m MonitorManager) Update(ctx context.Context, args []string) error {
	outputs, err := DetectOutputs()
	if err != nil {
		return err
	}

	rule, err := MatchRules(m.Config.Rules, outputs)
	if err != nil {
		return err
	}

	slog.Info("rule matches", "rule", rule.Name)
	return ApplyRule(ctx, outputs, rule, m.Config.ExecuteAfter)
}

func (m MonitorManager) Watch(ctx context.Context, args []string) error {
	done := make(chan struct{})
	defer close(done)

	ch := make(chan Event)
	go subscribeXEvents(ch, done)

	slog.Info("successfully subscribed to X RANDR change events")

	var tickerCh <-chan time.Time
	if opts.PollInterval > 0 {
		tickerCh = time.NewTicker(opts.PollInterval).C
	}

	var backoffCh <-chan time.Time
	var disablePoll bool
	var eventReceived bool

	var lastRule Rule
	var lastOutputs Outputs
	for {
		if !disablePoll {
			var outputs Outputs
			var err error

			if eventReceived || opts.ActivePoll {
				outputs, err = DetectOutputs()
				eventReceived = false
			} else {
				outputs, err = GetOutputs()
			}

			if err != nil {
				return fmt.Errorf("detecting outputs: %w", err)
			}

			// disable outputs which have a changed display
			var off Outputs
			for _, o := range outputs {
				for _, last := range lastOutputs {
					if o.Name != last.Name {
						continue
					}

					if last.Active() && !o.Active() {
						slog.Info("monitor not active any more, disabling output", "output", o.Name)
						off = append(off, o)
						continue
					}

					if o.Active() && o.MonitorID != last.MonitorID {
						slog.Info("monitor has changed, disabling output", "output", o.Name)
						off = append(off, o)
						continue
					}
				}
			}

			if len(off) > 0 {
				slog.Info("disable outputs", "outputs", off)

				cmd, err := DisableOutputs(off)
				if err != nil {
					return fmt.Errorf("disabling outputs: %w", err)
				}

				// forget the last rule set, something has changed for sure
				lastRule = Rule{}

				if err := RunCommand(ctx, cmd); err != nil {
					slog.Error("failed to disable outputs", "err", err)
				}

				// refresh outputs again
				outputs, err = GetOutputs()
				if err != nil {
					return fmt.Errorf("detecting outputs after disabling: %w", err)
				}

				slog.Info("new outputs after disable", "outputs", outputs)
			}

			rule, err := MatchRules(m.Config.Rules, outputs)
			if err != nil {
				return fmt.Errorf("matching rules: %w", err)
			}

			if rule.Name != lastRule.Name {
				slog.Info("new rule found", "rule", rule.Name, "outputs", outputs)

				err = ApplyRule(ctx, outputs, rule, m.Config.ExecuteAfter)
				if err != nil {
					return fmt.Errorf("applying rules: %w", err)
				}

				lastRule = rule

				if opts.Pause > 0 {
					slog.Info(fmt.Sprintf("disable polling for %d", opts.Pause))
					disablePoll = true
					backoffCh = time.After(opts.Pause)
				}

				// refresh outputs for next cycle
				outputs, err = GetOutputs()
				if err != nil {
					return fmt.Errorf("refreshing outputs: %w", err)
				}
			}

			lastOutputs = outputs
		}

		select {
		case ev := <-ch:
			slog.Info("new RANDR change event received", "event", ev.Event.String())
			if ev.Error != nil {
				return fmt.Errorf("RANDR change event contains error: %w", ev.Error)
			}

			eventReceived = true
		case <-tickerCh:
		case <-backoffCh:
			slog.Info("reenable polling")
			backoffCh = nil
			disablePoll = false
		case <-ctx.Done():
			return nil
		}
	}
}

func ApplyRule(ctx context.Context, outputs Outputs, rule Rule, execAfter []string) error {
	var cmds []*exec.Cmd

	switch {
	case rule.ConfigureSingle != nil, len(rule.ConfigureRow) > 0, len(rule.ConfigureColumn) > 0:
		var err error
		cmds, err = BuildCommandOutputRow(rule, outputs)
		if err != nil {
			return err
		}
	case rule.ConfigureCommand != "":
		cmds = []*exec.Cmd{exec.Command("sh", "-c", rule.ConfigureCommand)}
	default:
		return fmt.Errorf("no output configuration for rule %s", rule.Name)
	}

	after := append(execAfter, rule.ExecuteAfter...)
	for _, cmd := range after {
		cmds = append(cmds, exec.Command("sh", "-c", cmd))
	}

	for _, cmd := range cmds {
		if err := RunCommand(ctx, cmd); err != nil {
			slog.Error("executing command for rule failed", "rule", rule.Name, "err", err)
		}
	}

	return nil
}

func MatchRules(rules []Rule, outputs Outputs) (Rule, error) {
	for _, rule := range rules {
		if rule.Match(outputs) {
			return rule, nil
		}
	}

	return Rule{}, nil
}

// RunCommand runs the given command or prints the arguments to stdout if opts.DryRun is true.
func RunCommand(ctx context.Context, cmd *exec.Cmd) error {
	if opts.DryRun {
		slog.Info("dry-run command", "commad", strings.Join(cmd.Args, " "))
		return nil
	}

	slog.Info("running command", "command", fmt.Sprintf("%s %s", cmd.Path, strings.Join(cmd.Args, " ")))
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if opts.Verbose {
		cmd.Stdout = os.Stdout
	}

	return cmd.Run()
}

func subscribeXEvents(ch chan<- Event, done <-chan struct{}) {
	X, err := xgb.NewConn()
	if err != nil {
		ch <- Event{Error: err}
		return
	}

	defer X.Close()
	if err = randr.Init(X); err != nil {
		ch <- Event{Error: err}
		return
	}

	root := xproto.Setup(X).DefaultScreen(X).Root

	eventMask := randr.NotifyMaskScreenChange |
		randr.NotifyMaskCrtcChange |
		randr.NotifyMaskOutputChange |
		randr.NotifyMaskOutputProperty

	err = randr.SelectInputChecked(X, root, uint16(eventMask)).Check()
	if err != nil {
		ch <- Event{Error: err}
		return
	}

	for {
		ev, err := X.WaitForEvent()
		select {
		case ch <- Event{Event: ev, Error: err}:
		case <-time.After(eventSendTimeout):
			continue
		case <-done:
			return
		}
		if err != nil {
			slog.Error("failed to listen X events", "err", err)
			time.Sleep(100 * time.Millisecond)
		}
	}
}
