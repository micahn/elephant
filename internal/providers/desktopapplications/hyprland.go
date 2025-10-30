package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
)

type Hyprland struct{}

func (Hyprland) GetWorkspace() string {

	cmd := exec.Command("hyprctl", "activeworkspace")
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error(Name, "hyprlandworkspaces", err)
		return ""
	}
	workspaceID := strings.Split(string(out), " ")[2]
	return workspaceID
}

func (c Hyprland) MoveToWorkspace(workspace, initialWMClass string) {
	if initialWMClass == "" {
		slog.Error(Name, "movetoworkspace", ".desktopfile has no StartupWMClass")
		return

	}
	xdgRuntimeDir := os.Getenv("XDG_RUNTIME_DIR")
	hyperlandInstanceSignature := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")
	if xdgRuntimeDir == "" || hyperlandInstanceSignature == "" {
		panic("xdgRuntimeDir is Null or hyprland is not running!")
	}

	socketpath := fmt.Sprintf("%s/hypr/%s/.socket2.sock", xdgRuntimeDir, hyperlandInstanceSignature)

	conn, err := net.Dial("unix", socketpath)
	if err != nil {
		slog.Error(Name, "unix socket", err)
	}
	defer conn.Close()
	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {

		event := string(scanner.Bytes())
		if strings.HasPrefix(event, "openwindow>>") {
			windowinfo := strings.Split(strings.Split(event, ">>")[1], ",")
			if windowinfo[2] == initialWMClass && windowinfo[1] != workspace {
				bashline := fmt.Sprintf("hyprctl dispatch movetoworkspacesilent %s,address:0x%s", workspace, windowinfo[0])
				slog.Info(Name, "movetoworkspace", bashline)
				cmd := exec.Command("/usr/bin/env", "bash", "-c", bashline)
				out, err := cmd.CombinedOutput()
				if err != nil {
					slog.Error(Name, "movetoworkspace", out)
				}
				slog.Info(Name, "movetoworkspace", string(out))
				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error(Name, "monitor", err)
		return
	}

}
