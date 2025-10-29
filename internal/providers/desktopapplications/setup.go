package main

import (
	"bufio"
	"bytes"
	_ "embed"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/abenz1267/elephant/v2/internal/util/windows"
	"github.com/abenz1267/elephant/v2/pkg/common"
	"github.com/abenz1267/elephant/v2/pkg/common/history"
)

type DesktopFile struct {
	Data
	Actions []Data
}

var (
	Name       = "desktopapplications"
	NamePretty = "Desktop Applications"
	h          = history.Load(Name)
	pins       = loadpinned()
	config     *Config
	br         = []*regexp.Regexp{}
	wmi        WMIntegration
)

type WMIntegration interface {
	GetWorkspace() string
	MoveToWorkspace(workspace, initialWMClass string)
}

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
			return after
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

func (Niri) MoveToWorkspace(workspace, initialWMClass string) {
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

		if e.WindowOpenedOrChanged != nil && e.WindowOpenedOrChanged.Window.AppID == initialWMClass && e.WindowOpenedOrChanged.Window.Layout.PosInScrollingLayout != nil {
			fmt.Println("HERE")
			cmd := exec.Command("niri", "msg", "action", "move-window-to-workspace", workspace)
			out, err := cmd.CombinedOutput()
			if err != nil {
				slog.Error(Name, "nirimovetoworkspace", out)
			}
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

//go:embed README.md
var readme string

type Config struct {
	common.Config           `koanf:",squash"`
	LaunchPrefix            string            `koanf:"launch_prefix" desc:"overrides the default app2unit or uwsm prefix, if set." default:""`
	Locale                  string            `koanf:"locale" desc:"to override systems locale" default:""`
	ActionMinScore          int               `koanf:"action_min_score" desc:"min score for actions to be shown" default:"20"`
	ShowActions             bool              `koanf:"show_actions" desc:"include application actions, f.e. 'New Private Window' for Firefox" default:"false"`
	ShowGeneric             bool              `koanf:"show_generic" desc:"include generic info when show_actions is true" default:"true"`
	ShowActionsWithoutQuery bool              `koanf:"show_actions_without_query" desc:"show application actions, if the search query is empty" default:"false"`
	History                 bool              `koanf:"history" desc:"make use of history for sorting" default:"true"`
	HistoryWhenEmpty        bool              `koanf:"history_when_empty" desc:"consider history when query is empty" default:"false"`
	OnlySearchTitle         bool              `koanf:"only_search_title" desc:"ignore keywords, comments etc from desktop file when searching" default:"false"`
	IconPlaceholder         string            `koanf:"icon_placeholder" desc:"placeholder icon for apps without icon" default:"applications-other"`
	Aliases                 map[string]string `koanf:"aliases" desc:"setup aliases for applications. Matched aliases will always be placed on top of the list. Example: 'ffp' => '<identifier>'. Check elephant log output when activating an item to get its identifier." default:""`
	Blacklist               []string          `koanf:"blacklist" desc:"blacklist desktop files from being parsed. Regexp." default:"<empty>"`
	WindowIntegration       bool              `koanf:"window_integration" desc:"will enable window integration, meaning focusing an open app instead of opening a new instance" default:"false"`
	WMIngegration           bool              `koanf:"wm_integration" desc:"enhances the experience based on the window manager in use. Currently Niri only." default:"true"`
}

func loadpinned() []string {
	pinned := []string{}

	file := common.CacheFile(fmt.Sprintf("%s_pinned.gob", Name))

	if common.FileExists(file) {
		f, err := os.ReadFile(file)
		if err != nil {
			slog.Error("pinned", "load", err)
		} else {
			decoder := gob.NewDecoder(bytes.NewReader(f))

			err = decoder.Decode(&pinned)
			if err != nil {
				slog.Error("pinned", "decoding", err)
			}
		}
	}

	return pinned
}

func Setup() {
	start := time.Now()
	config = &Config{
		Config: common.Config{
			Icon:     "applications-other",
			MinScore: 30,
		},
		ActionMinScore:          20,
		OnlySearchTitle:         false,
		ShowActions:             false,
		ShowGeneric:             true,
		ShowActionsWithoutQuery: false,
		History:                 true,
		WMIngegration:           true,
		HistoryWhenEmpty:        false,
		IconPlaceholder:         "applications-other",
		Aliases:                 map[string]string{},
		WindowIntegration:       false,
	}

	common.LoadConfig(Name, config)

	parseRegexp()
	loadFiles()

	if config.WindowIntegration {
		if !windows.IsSetup {
			windows.Init()
		}
	}

	switch os.Getenv("XDG_CURRENT_DESKTOP") {
	case "niri":
		wmi = Niri{}
	}

	config.WMIngegration = wmi != nil

	slog.Info(Name, "desktop files", len(files), "time", time.Since(start))
}

func parseRegexp() {
	for _, v := range config.Blacklist {
		r, err := regexp.Compile(v)
		if err != nil {
			log.Panic(err)
		}

		br = append(br, r)
	}
}

func Icon() string {
	return config.Icon
}
