// Package symbols provides symbols/emojis.
package main

import (
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"time"

	_ "embed"

	"github.com/abenz1267/elephant/v2/internal/util"
	"github.com/abenz1267/elephant/v2/internal/util/windows"
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

func Setup() {
	start := time.Now()

	if !windows.IsSetup {
		windows.Init()
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

	err := windows.FocusWindow(i)
	if err != nil {
		slog.Error(Name, "activate", err)
	}
}

func Query(conn net.Conn, query string, _ bool, exact bool) []*pb.QueryResponse_Item {
	start := time.Now()

	entries := []*pb.QueryResponse_Item{}

	windows, err := windows.GetWindowList()
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

func calcScore(q string, d *windows.Window, exact bool) (string, int32, []int32, int32, bool) {
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
