// Package symbols provides symbols/emojis.
package main

/*
#cgo LDFLAGS: -lwayland-client
#include "window_manager.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"
	"unsafe"

	_ "embed"

	"github.com/abenz1267/elephant/v2/internal/util"
	"github.com/abenz1267/elephant/v2/pkg/common"
	"github.com/abenz1267/elephant/v2/pkg/pb/pb"
)

var (
	Name       = "windows"
	NamePretty = "Windows"
)

//go:embed README.md
var readme string

type Config struct {
	common.Config `koanf:",squash"`
	Delay         int `koanf:"delay" desc:"delay in ms before focusing to avoid potential focus issues" default:"100"`
}

var config *Config

type Window struct {
	ID    int
	Title string
	AppID string
}

func initWindowManager() error {
	result := C.init_window_manager()
	if result != 0 {
		return fmt.Errorf("failed to initialize window manager")
	}
	return nil
}

func getWindowList() ([]Window, error) {
	windowList := C.get_window_list()
	if windowList == nil {
		return nil, fmt.Errorf("failed to get window list - window manager may not be initialized or no Wayland compositor found")
	}

	count := int(windowList.count)
	if count == 0 {
		return []Window{}, nil
	}

	if windowList.windows == nil {
		return nil, fmt.Errorf("window list array is null")
	}

	windows := make([]Window, count)

	// Access the C array safely
	windowArray := (*[1000]C.window_info_t)(unsafe.Pointer(windowList.windows))

	for i := range count {
		window := windowArray[i]

		var title, appID string
		if window.title != nil {
			title = C.GoString(window.title)
		}
		if window.app_id != nil {
			appID = C.GoString(window.app_id)
		}

		windows[i] = Window{
			ID:    int(window.id),
			Title: title,
			AppID: appID,
		}
	}

	return windows, nil
}

func focusWindow(windowID int) error {
	result := C.focus_window(C.int(windowID))
	switch result {
	case 0:
		return nil
	case -1:
		return fmt.Errorf("window with ID %d not found", windowID)
	case -2:
		return fmt.Errorf("no seat available for focusing (may need input device)")
	default:
		return fmt.Errorf("failed to focus window with ID %d (error %d)", windowID, int(result))
	}
}

func Setup() {
	start := time.Now()

	if err := initWindowManager(); err != nil {
		slog.Error(Name, "init", err)
		return
	}

	config = &Config{
		Config: common.Config{
			Icon:     "view-restore",
			MinScore: 20,
		},
		Delay: 100,
	}

	common.LoadConfig(Name, config)

	slog.Info(Name, "loaded", time.Since(start))
}

func PrintDoc() {
	fmt.Println(readme)
	fmt.Println()
	util.PrintConfig(Config{}, Name)
}

const (
	ActionFocus = "focus"
)

func Activate(identifier, action string, query string, args string) {
	time.Sleep(time.Duration(config.Delay) * time.Millisecond)

	i, _ := strconv.Atoi(identifier)

	err := focusWindow(i)
	if err != nil {
		slog.Error(Name, "activate", err)
	}
}

func Query(conn net.Conn, query string, _ bool, exact bool) []*pb.QueryResponse_Item {
	start := time.Now()

	entries := []*pb.QueryResponse_Item{}

	windows, err := getWindowList()
	if err != nil {
		slog.Error(Name, "query", err)
		return entries
	}

	for _, window := range windows {
		e := &pb.QueryResponse_Item{
			Identifier: fmt.Sprintf("%d", window.ID),
			Text:       window.Title,
			Subtext:    window.AppID,
			Actions:    []string{ActionFocus},
			Provider:   Name,
			Icon:       config.Icon,
		}

		if query != "" {
			matched, score, pos, start, ok := calcScore(query, &window, exact)

			if ok {
				field := "text"
				e.Score = score

				if matched != window.Title {
					field = "subtext"
				}

				e.Fuzzyinfo = &pb.QueryResponse_Item_FuzzyInfo{
					Start:     start,
					Field:     field,
					Positions: pos,
				}
			}
		}

		if query == "" || e.Score > config.MinScore {
			entries = append(entries, e)
		}
	}

	slog.Info(Name, "queryresult", len(entries), "time", time.Since(start))

	return entries
}

func Icon() string {
	return config.Icon
}

func calcScore(q string, d *Window, exact bool) (string, int32, []int32, int32, bool) {
	var scoreRes int32
	var posRes []int32
	var startRes int32
	var match string
	var modifier int32

	toSearch := []string{d.Title, d.AppID}

	for k, v := range toSearch {
		score, pos, start := common.FuzzyScore(q, v, exact)

		if score > scoreRes {
			scoreRes = score
			posRes = pos
			startRes = start
			match = v
			modifier = int32(k)
		}
	}

	if scoreRes == 0 {
		return "", 0, nil, 0, false
	}

	scoreRes = max(scoreRes-min(modifier*5, 50)-startRes, 10)

	return match, scoreRes, posRes, startRes, true
}
