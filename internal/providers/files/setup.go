package main

import (
	"bytes"
	"crypto/md5"
	_ "embed"
	"encoding/hex"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/abenz1267/elephant/v2/internal/util"
	"github.com/abenz1267/elephant/v2/pkg/common"
	"github.com/djherbis/times"
	"github.com/fsnotify/fsnotify"
)

//go:embed README.md
var readme string

var (
	Name       = "files"
	NamePretty = "Files"
	config     *Config
)

type Config struct {
	common.Config `koanf:",squash"`
	LaunchPrefix  string   `koanf:"launch_prefix" desc:"overrides the default app2unit or uwsm prefix, if set." default:""`
	IgnoredDirs   []string `koanf:"ignored_dirs" desc:"ignore these directories" default:""`
}

func Setup() {
	start := time.Now()

	err := openDB()
	if err != nil {
		slog.Error(Name, "setup", err)
		return
	}

	config = &Config{
		Config: common.Config{
			Icon:     "folder",
			MinScore: 20,
		},
		LaunchPrefix: "",
	}

	common.LoadConfig(Name, config)

	home, _ := os.UserHomeDir()
	cmd := exec.Command("fd", ".", home, "--ignore-vcs", "--type", "file", "--type", "directory")

	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error(Name, "files", err)
		os.Exit(1)
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}

	deleteChan := make(chan string)

	go func() {
		timer := time.NewTimer(time.Second * 5)
		do := false
		toDelete := []string{}

		for {
			select {
			case path := <-deleteChan:
				timer.Reset(time.Second * 2)
				toDelete = append(toDelete, path)
				do = true
			case <-timer.C:
				if do {
					slices.Sort(toDelete)
					toDelete = slices.Compact(toDelete)

					for _, path := range toDelete {
						deleteFileByPath(path)
					}

					toDelete = []string{}
					do = false
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}

				if event.Op == fsnotify.Remove || event.Op == fsnotify.Rename {
					deleteChan <- event.Name
				}

				if info, err := times.Stat(event.Name); err == nil {
					fileInfo, err := os.Stat(event.Name)
					if err == nil {
						path := event.Name

						if fileInfo.IsDir() {
							path = path + "/"
							watcher.Add(path)
						}

						md5 := md5.Sum([]byte(path))
						md5str := hex.EncodeToString(md5[:])

						if val := getFile(md5str); val != nil {
							val.Changed = info.ChangeTime()
							putFile(*val)
						} else {
							putFile(File{
								Identifier: md5str,
								Path:       path,
								Changed:    info.ChangeTime(),
							})
						}
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return
				}
			}
		}
	}()

	toPut := []File{}

outer:
	for v := range bytes.Lines(out) {
		if len(v) > 0 {
			path := strings.TrimSpace(string(v))

			for _, v := range config.IgnoredDirs {
				if strings.HasPrefix(path, v) {
					continue outer
				}
			}

			if strings.HasSuffix(path, "/") {
				watcher.Add(path)
			}

			if info, err := times.Stat(path); err == nil {
				diff := start.Sub(info.ChangeTime())

				md5 := md5.Sum([]byte(path))
				md5str := hex.EncodeToString(md5[:])

				f := File{
					Identifier: md5str,
					Path:       path,
					Changed:    time.Time{},
				}

				res := 3600 - diff.Seconds()
				if res > 0 {
					f.Changed = info.ChangeTime()
				}

				toPut = append(toPut, f)
			}
		}
	}

	putFileBatch(toPut)

	slog.Info(Name, "time", time.Since(start))
}

func PrintDoc() {
	fmt.Println(readme)
	fmt.Println()
	util.PrintConfig(Config{}, Name)
}

func Icon() string {
	return config.Icon
}
