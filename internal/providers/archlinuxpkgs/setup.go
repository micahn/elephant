package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/exec"
	"runtime/debug"
	"slices"
	"strings"
	"time"

	"github.com/abenz1267/elephant/v2/internal/util"
	"github.com/abenz1267/elephant/v2/pkg/common"
	"github.com/abenz1267/elephant/v2/pkg/pb/pb"
)

var (
	Name          = "archlinuxpkgs"
	NamePretty    = "Arch Linux Packages"
	config        *Config
	packages      = map[string]Package{}
	installed     = []string{}
	installedOnly = false
	cacheFile     = common.CacheFile("archlinuxpkgs.json")
)

//go:embed README.md
var readme string

const (
	ActionInstall       = "install"
	ActionClearCache    = "clear_cache"
	ActionVisitURL      = "visit_url"
	ActionRefresh       = "refresh"
	ActionRemove        = "remove"
	ActionShowInstalled = "show_installed"
	ActionShowAll       = "show_all"
)

type Config struct {
	common.Config        `koanf:",squash"`
	CommandInstall       string `koanf:"command_install" desc:"default command for AUR packages to install. supports %VALUE%." default:"yay -S %VALUE%"`
	CommandRemove        string `koanf:"command_remove" desc:"default command to remove packages. supports %VALUE%." default:"sudo pacman -R %VALUE%"`
	AutoWrapWithTerminal bool   `koanf:"auto_wrap_with_terminal" desc:"automatically wraps the command with terminal" default:"true"`
}

type Package struct {
	Name           string  `json:"name,omitempty"`
	Description    string  `json:"description,omitempty"`
	Repository     string  `json:"repository,omitempty"`
	Version        string  `json:"version,omitempty"`
	Installed      bool    `json:"installed,omitempty"`
	FullInfo       string  `json:"full_info,omitempty"`
	URL            string  `json:"url,omitempty"`
	URLPath        string  `json:"url_path,omitempty"`
	Maintainer     string  `json:"maintainer,omitempty"`
	Submitter      string  `json:"submitter,omitempty"`
	NumVotes       int     `json:"num_votes,omitempty"`
	Popularity     float64 `json:"popularity,omitempty"`
	FirstSubmitted int64   `json:"first_submitted,omitempty"`
	LastModified   int64   `json:"last_modified,omitempty"`
}

type AURPackage struct {
	Name           string  `json:"name,omitempty"`
	Description    string  `json:"description,omitempty"`
	Version        string  `json:"version,omitempty"`
	URL            string  `json:"url,omitempty"`
	URLPath        string  `json:"url_path,omitempty"`
	Maintainer     string  `json:"maintainer,omitempty"`
	Submitter      string  `json:"submitter,omitempty"`
	NumVotes       int     `json:"num_votes,omitempty"`
	Popularity     float64 `json:"popularity,omitempty"`
	FirstSubmitted int64   `json:"first_submitted,omitempty"`
	LastModified   int64   `json:"last_modified,omitempty"`
}

func (a AURPackage) toFullInfo() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("%-*s: %s\n", 15, "Name", a.Name))
	b.WriteString(fmt.Sprintf("%-*s: %s\n", 15, "Description", a.Description))
	b.WriteString(fmt.Sprintf("%-*s: %s\n", 15, "Version", a.Version))
	b.WriteString(fmt.Sprintf("%-*s: %s\n", 15, "URL", a.URL))
	b.WriteString(fmt.Sprintf("%-*s: %s\n", 15, "URL-Path", a.URLPath))
	b.WriteString(fmt.Sprintf("%-*s: %s\n", 15, "Maintainer", a.Maintainer))
	b.WriteString(fmt.Sprintf("%-*s: %s\n", 15, "Submitter", a.Submitter))
	b.WriteString(fmt.Sprintf("%-*s: %d\n", 15, "Submitted", a.FirstSubmitted))
	b.WriteString(fmt.Sprintf("%-*s: %d\n", 15, "Votes", a.NumVotes))
	b.WriteString(fmt.Sprintf("%-*s: %.2f\n", 15, "Popularity", a.Popularity))
	b.WriteString(fmt.Sprintf("%-*s: %d\n", 15, "Modified", a.LastModified))

	return b.String()
}

func detectHelper() string {
	helpers := []string{"paru", "yay"}
	for _, h := range helpers {
		if _, err := exec.LookPath(h); err == nil {
			return h
		}
	}
	return "sudo pacman"
}

var cacheChan = make(chan struct{})

func clearCache() {
	timer := time.NewTimer(time.Second * 30)
	do := false

	for {
		select {
		case <-cacheChan:
			timer.Reset(time.Second * 30)
			do = true
		case <-timer.C:
			if do {
				packages = make(map[string]Package)
				debug.FreeOSMemory()
				do = false
			}
		}
	}
}

func Setup() {
	helper := detectHelper()

	config = &Config{
		Config: common.Config{
			Icon:     "applications-internet",
			MinScore: 20,
		},
		CommandInstall:       fmt.Sprintf("%s -S %s", helper, "%VALUE%"),
		CommandRemove:        fmt.Sprintf("%s -R %s", helper, "%VALUE%"),
		AutoWrapWithTerminal: true,
	}

	common.LoadConfig(Name, config)

	setup()
	go clearCache()
}

func setup() {
	getInstalled()
	getOfficialPkgs()
	setupAURPkgs()

	b, err := json.Marshal(packages)
	if err != nil {
		slog.Error(Name, "cache marshal", err)
		return
	}

	os.Remove(cacheFile)
	_ = os.WriteFile(cacheFile, b, 0o600)

	packages = make(map[string]Package)
	debug.FreeOSMemory()
}

func Available() bool {
	return true
}

func PrintDoc() {
	fmt.Println(readme)
	fmt.Println()
	util.PrintConfig(Config{}, Name)
}

func Activate(single bool, identifier, action string, query string, args string, format uint8, conn net.Conn) {
	defer func() {
		packages = make(map[string]Package)
		debug.FreeOSMemory()
	}()

	switch action {
	case ActionVisitURL:
		run := strings.TrimSpace(fmt.Sprintf("%s xdg-open '%s'", common.LaunchPrefix(""), packages[identifier].URL))
		cmd := exec.Command("sh", "-c", run)

		err := cmd.Start()
		if err != nil {
			slog.Error(Name, "activate", err, "action", action)
		} else {
			go func() {
				cmd.Wait()
			}()
		}

		return
	case ActionRefresh:
		setup()
		return
	case ActionShowAll:
		installedOnly = false
		return
	case ActionShowInstalled:
		installedOnly = true
		return
	}

	name := packages[identifier].Name
	var pkgcmd string

	switch action {
	case ActionInstall:
		pkgcmd = config.CommandInstall
	case ActionRemove:
		pkgcmd = config.CommandRemove
	default:
		slog.Error(Name, "activate", fmt.Sprintf("unknown action: %s", action))
		return
	}

	pkgcmd = strings.ReplaceAll(pkgcmd, "%VALUE%", name)
	toRun := common.WrapWithTerminal(pkgcmd)

	if !config.AutoWrapWithTerminal {
		toRun = pkgcmd
	}

	cmd := exec.Command("sh", "-c", toRun)
	err := cmd.Start()
	if err != nil {
		slog.Error(Name, "activate", err)
	} else {
		go func() {
			cmd.Wait()
		}()
	}
}

func Query(conn net.Conn, query string, single bool, exact bool, _ uint8) []*pb.QueryResponse_Item {
	cacheChan <- struct{}{}

	entries := []*pb.QueryResponse_Item{}

	if len(packages) == 0 {
		b, _ := os.ReadFile(cacheFile)
		json.Unmarshal(b, &packages)
	}

	for k, v := range packages {
		score, positions, s := common.FuzzyScore(query, v.Name, exact)

		score2, positions2, s2 := common.FuzzyScore(query, v.Description, exact)

		if score2 > score {
			score = score2 / 2
			positions = positions2
			s = s2
		}

		if (score > config.MinScore || query == "") && (!installedOnly || (installedOnly && v.Installed)) {
			state := []string{}
			a := []string{}

			if v.Installed {
				state = append(state, "installed")
				a = append(a, ActionRemove)
			} else {
				state = append(state, "available")
				a = append(a, ActionInstall)
			}

			if v.URL != "" {
				a = append(a, "visit_url")
			}

			subtext := fmt.Sprintf("[%s]", strings.ToLower(v.Repository))
			if v.Installed {
				subtext = fmt.Sprintf("[%s] [installed]", strings.ToLower(v.Repository))
			}

			entries = append(entries, &pb.QueryResponse_Item{
				Identifier:  k,
				Text:        v.Name,
				Type:        pb.QueryResponse_REGULAR,
				Subtext:     subtext,
				Provider:    Name,
				State:       state,
				Actions:     a,
				Score:       score,
				Preview:     v.FullInfo,
				PreviewType: util.PreviewTypeText,
				Fuzzyinfo: &pb.QueryResponse_Item_FuzzyInfo{
					Start:     s,
					Field:     "text",
					Positions: positions,
				},
			})
		}
	}

	if query == "" {
		slices.SortFunc(entries, func(a, b *pb.QueryResponse_Item) int {
			return strings.Compare(a.Text, b.Text)
		})
	}

	return entries
}

func Icon() string {
	return config.Icon
}

func HideFromProviderlist() bool {
	return config.HideFromProviderlist
}

func State(provider string) *pb.ProviderStateResponse {
	actions := []string{ActionRefresh}

	if installedOnly {
		actions = append(actions, ActionShowAll)
	} else {
		actions = append(actions, ActionShowInstalled)
	}

	return &pb.ProviderStateResponse{
		Actions: actions,
	}
}

func getOfficialPkgs() {
	cmd := exec.Command("pacman", "-Si")

	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error(Name, "pacman", err)
	}

	var data strings.Builder
	e := Package{}

	for line := range strings.Lines(string(out)) {
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.SplitN(strings.TrimSpace(line), ":", 2)

		if len(parts) == 2 {
			data.WriteString(fmt.Sprintf("%-*s: %s\n", 15, parts[0], strings.TrimSpace(parts[1])))
		} else {
			data.WriteString(fmt.Sprintf("%-*s", 17, line))
		}

		switch {
		case strings.HasPrefix(line, "Repository"):
			e.Repository = strings.TrimSpace(strings.Split(line, ":")[1])
		case strings.HasPrefix(line, "Name"):
			e.Name = strings.TrimSpace(strings.Split(line, ":")[1])
			e.Installed = slices.Contains(installed, e.Name)
		case strings.HasPrefix(line, "Description"):
			e.Description = strings.TrimSpace(strings.Split(line, ":")[1])
		case strings.HasPrefix(line, "Version"):
			e.Version = strings.TrimSpace(strings.Split(line, ":")[1])
		case strings.HasPrefix(line, "URL"):
			e.URL = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}

		if strings.Contains(line, "Validated By") {
			e.FullInfo = data.String()
			packages[e.Name] = e
			e = Package{}
			data.Reset()
		}
	}
}

func setupAURPkgs() {
	resp, err := http.Get("https://aur.archlinux.org/packages-meta-v1.json.gz")
	if err != nil {
		slog.Error(Name, "aurdownload", err)
		return
	}
	defer resp.Body.Close()

	decoder := json.NewDecoder(resp.Body)

	var aurPackages []AURPackage

	err = decoder.Decode(&aurPackages)
	if err != nil {
		slog.Error(Name, "jsondecode", err)
		return
	}

	for _, pkg := range aurPackages {
		packages[pkg.Name] = Package{
			Name:        pkg.Name,
			Description: pkg.Description,
			Version:     pkg.Version,
			Repository:  "aur",
			Installed:   slices.Contains(installed, pkg.Name),
			URL:         pkg.URL,
			FullInfo:    pkg.toFullInfo(),
		}
	}
}

func getInstalled() {
	installed = []string{}

	cmd := exec.Command("pacman", "-Qe")
	out, err := cmd.CombinedOutput()
	if err != nil {
		slog.Error(Name, "installed", err)
	}

	for line := range strings.Lines(string(out)) {
		fields := strings.Fields(line)
		if len(fields) > 0 {
			installed = append(installed, fields[0])
		}
	}
}
