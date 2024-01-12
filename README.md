[![Build Status](https://github.com/fd0/grobi/workflows/test/badge.svg)](https://github.com/fd0/grobi/actions?query=workflow%3Atest)

# grobi

This program watches for changes in the available outputs (e.g. when a monitor
is connected/disconnected) and will automatically configure the currently used
outputs via RANDR according to configurable profiles.

# Installation

Grobi requires Go version 1.11 or newer to compile. To build grobi, run the
following command:

```shell
$ go build
```

Afterwards please find a binary of grobi in the current directory:
```
$ ./grobi --help
Usage of ./grobi:
  -active-poll
        Force xrandr to re-detect outputs during polling.
  -config string
        The path to a config file.
  -dry-run
        Enable dry-run mode: print commands instead of running them.
  -interval duration
        Duration between polls. Set to zero to disable polling. (default 2s)
  -pause duration
        Number of seconds to pause after a change was executed.
  -verbose
        Enable verbose logging

Available commands:
        version Display the version.
        apply   Apply a rule to configure the outputs accordingly.
        rules   List the configured rules.
        show    Show monitors and IDs.
        update  Update outputs config.
        watch   Watch for XRANDR changes.

```

# Configuration

Have a look at the sample configuration file provided at
[`doc/grobi.conf`](doc/grobi.conf). By default, `grobi` uses the [XDG directory
standard](https://standards.freedesktop.org/basedir-spec/basedir-spec-latest.html)
to search for the config file. Most users will probably put the config file to
`~/.config/grobi.conf`.

If you have any questions, please open an issue on GitHub.

There is also a [sample systemd](doc/grobi.service) unit file you can run as a
user. This requires that the `PATH` and `DISPLAY` environment variables can be
accessed, so run the following command in e.g. your `~/.xsession` file just
before starting the window manager:

```
systemctl --user import-environment DISPLAY PATH
```

Run the command once to import the environment for the current session, then
execute the following commands to install and start the unit:

```
mkdir ~/.config/systemd/user
cp doc/grobi.service ~/.config/systemd/user
systemctl --user enable grobi
systemctl --user start grobi
```

You can then use `systemctl` to check the current status:

```
systemctl --user status grobi
```

# Compatibility

Grobi follows [Semantic Versioning](http://semver.org) to clearly define which
versions are compatible. The configuration file and command-line parameters and
user-interface are considered the "Public API" in the sense of Semantic
Versioning.

# Development

## Release New Version

Rough steps for releasing a new version:
 * Update version number in `cmd_version.go`, remove the `-dev` suffix
 * Commit and tag version
 * Add `-dev` suffix to version in `cmd_version.go`
