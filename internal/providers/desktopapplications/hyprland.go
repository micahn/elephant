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

	workspaceID := strings.Fields(string(out))[2]

	return workspaceID
}

func (c Hyprland) MoveToWorkspace(workspace, initialWMClass string) {
	runtimeDir := os.Getenv("XDG_RUNTIME_DIR")
	instanceSignature := os.Getenv("HYPRLAND_INSTANCE_SIGNATURE")

	if runtimeDir == "" || instanceSignature == "" {
		slog.Error(Name, "hyprlandmovetoworkspace", "XDG_RUNTIME_DIR or HYPRLAND_INSTANCE_SIGNATURE missing")

		return
	}

	socket := fmt.Sprintf("%s/hypr/%s/.socket2.sock", runtimeDir, instanceSignature)

	conn, err := net.Dial("unix", socket)
	if err != nil {
		slog.Error(Name, "unix socket", err)
	}
	defer conn.Close()

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		event := scanner.Text()

		if strings.HasPrefix(event, "openwindow>>") {
			windowinfo := strings.Split(strings.Split(event, ">>")[1], ",")

			if windowinfo[2] == initialWMClass && windowinfo[1] != workspace {
				cmd := exec.Command("sh", "-c", fmt.Sprintf("hyprctl dispatch movetoworkspacesilent %s,address:0x%s", workspace, windowinfo[0]))

				out, err := cmd.CombinedOutput()
				if err != nil {
					slog.Error(Name, "movetoworkspace", out)
				}

				return
			}
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error(Name, "monitor", err)
		return
	}
}
