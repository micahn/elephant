package common

import (
	"log/slog"
	"os"
	"os/exec"
)

var runPrefix = ""

func InitRunPrefix() {
	app2unit, err := exec.LookPath("app2unit")
	if err == nil && app2unit != "" {
		xdgTerminalExec, err := exec.LookPath("xdg-terminal-exec")
		if err == nil && xdgTerminalExec != "" {
			runPrefix = "app2unit"
			slog.Info("config", "runprefix", runPrefix)
			return
		}
	}

	uwsm, err := exec.LookPath("uwsm")
	if err == nil {
		cmd := exec.Command(uwsm, "check", "is-active")
		err := cmd.Run()
		if err == nil {
			runPrefix = "uwsm app --"
			slog.Info("config", "runprefix", runPrefix)
			return
		}
	}

	spid := os.Getenv("SYSTEMD_EXEC_PID")
	if spid != "" {
		systemdrun, err := exec.LookPath("systemd-run")
		if err == nil && systemdrun != "" {
			runPrefix = "systemd-run --user"
			slog.Info("config", "runprefix", runPrefix)
			return
		}
	}

	slog.Info("config", "runprefix", "<empty>")
}

func LaunchPrefix(override string) string {
	if override == "CLEAR" {
		return ""
	}

	if override != "" {
		return override
	}

	return runPrefix
}
