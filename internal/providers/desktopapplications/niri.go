package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
)

type Niri struct{}

func (Niri) GetWorkspace() string {
	cmd := exec.Command("niri", "msg", "workspaces")

	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error(Name, "niriworkspaces", err)
		return ""
	}

	for line := range strings.Lines(string(out)) {
		line = strings.TrimSpace(line)

		if after, ok := strings.CutPrefix(line, "*"); ok {
			return strings.TrimSpace(after)
		}
	}

	return ""
}

type OpenedOrChangedEvent struct {
	WindowOpenedOrChanged *struct {
		Window struct {
			ID     int    `json:"id"`
			AppID  string `json:"app_id"`
			Layout struct {
				PosInScrollingLayout []int `json:"pos_in_scrolling_layout"`
			} `json:"layout"`
		} `json:"window"`
	} `json:"WindowOpenedOrChanged,omitempty"`
}

func (c Niri) MoveToWorkspace(workspace, initialWMClass string) {
	cmd := exec.Command("niri", "msg", "-j", "event-stream")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error(Name, "monitor", err)
		return
	}

	if err := cmd.Start(); err != nil {
		slog.Error(Name, "monitor", err)
		return
	}

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		var e OpenedOrChangedEvent
		err := json.Unmarshal(scanner.Bytes(), &e)
		if err != nil {
			slog.Error(Name, "event unmarshal", err)
		}

		ws := c.GetWorkspace()

		if ws != workspace && e.WindowOpenedOrChanged != nil && e.WindowOpenedOrChanged.Window.AppID == initialWMClass && e.WindowOpenedOrChanged.Window.Layout.PosInScrollingLayout != nil {
			cmd := exec.Command("niri", "msg", "action", "move-window-to-workspace", workspace, "--window-id", fmt.Sprintf("%d", e.WindowOpenedOrChanged.Window.ID), "--focus", "false")
			out, err := cmd.CombinedOutput()
			if err != nil {
				slog.Error(Name, "nirimovetoworkspace", out)
			}

			return
		}
	}

	if err := scanner.Err(); err != nil {
		slog.Error(Name, "monitor", err)
		return
	}

	if err := cmd.Wait(); err != nil {
		slog.Error(Name, "monitor", err)
		return
	}
}
