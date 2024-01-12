package main

import "fmt"

// Rule is a rule to configure outputs.
type Rule struct {
	Name string

	OutputsConnected    []string `yaml:"outputs_connected,omitempty"`
	OutputsDisconnected []string `yaml:"outputs_disconnected,omitempty"`
	OutputsPresent      []string `yaml:"outputs_present,omitempty"`
	OutputsAbsent       []string `yaml:"outputs_absent,omitempty"`

	ConfigureRow     []*OutputConfig `yaml:"configure_row,omitempty"`
	ConfigureColumn  []*OutputConfig `yaml:"configure_column,omitempty"`
	ConfigureSingle  *OutputConfig   `yaml:"configure_single,omitempty"`
	ConfigureCommand string          `yaml:"configure_command,omitempty"`

	Primary      string   `yaml:"primary"`
	DisableOrder []string `yaml:"disable_order"`
	Atomic       bool     `yaml:"atomic"`
	ExecuteAfter []string `yaml:"execute_after"`
}

type OutputConfig struct {
	Name string `yaml:"name"`
	Mode string `yaml:"mode"`
	DPI  string `yaml:"dpi"`
}

func (cfg OutputConfig) String() string {
	return fmt.Sprintf("%s --mode %s --dpi %s", cfg.Name, cfg.Mode, cfg.DPI)
}

// Match returns true iff the rule matches for the given list of outputs.
func (r Rule) Match(outputs Outputs) bool {
	for _, name := range r.OutputsAbsent {
		if outputs.Present(name) {
			return false
		}
	}

	for _, name := range r.OutputsDisconnected {
		if outputs.Connected(name) {
			return false
		}
	}

	for _, name := range r.OutputsPresent {
		if !outputs.Present(name) {
			return false
		}
	}

	for _, name := range r.OutputsConnected {
		if !outputs.Connected(name) {
			return false
		}
	}

	return true
}
