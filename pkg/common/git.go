package common

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/abenz1267/elephant/pkg/common"
	"github.com/go-git/go-git/v6"
)

var gitMu sync.Mutex

func SetupGit(provider, url string) (string, *git.Worktree, *git.Repository) {
	gitMu.Lock()
	defer gitMu.Unlock()

	x := 0
	base := filepath.Base(url)
	folder := common.CacheFile(base)
	var w *git.Worktree
	var r *git.Repository

	for x < 15 {
		x++

		time.Sleep(1 * time.Second)

		slog.Info(provider, "gitsetup", "trying to setup git...")

		// clone
		if !common.FileExists(folder) {
			var err error

			url := url
			if strings.HasPrefix(url, "https://github.com/") {
				url = strings.Replace(url, "https://github.com/", "git@github.com:", 1)
			}

			r, err = git.PlainClone(folder, &git.CloneOptions{
				URL:               url,
				RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
			})
			if err != nil {
				slog.Debug(provider, "gitclone", err)
				continue
			}
		} else {
			var err error
			r, err = git.PlainOpen(folder)
			if err != nil {
				slog.Debug(provider, "gitclone", err)
				continue
			}
		}

		var err error

		w, err = r.Worktree()
		if err != nil {
			slog.Debug(provider, "gitpull", err)
			continue
		}

		err = w.Pull(&git.PullOptions{RemoteName: "origin"})
		if err != nil {
			slog.Info(provider, "gitpull", err)

			if err.Error() != "already up-to-date" {
				continue
			}
		}

		break
	}

	return folder, w, r
}

type PushData struct {
	provider string
	file     string
	w        *git.Worktree
	r        *git.Repository
}

var pushChan chan PushData

func init() {
	pushChan = make(chan PushData)

	go func() {
		timer := time.NewTimer(time.Second * 5)
		do := false

		var mu sync.Mutex
		work := make(map[string]PushData)

		for {
			select {
			case data := <-pushChan:
				mu.Lock()
				work[fmt.Sprintf("%s%s", data.provider, data.file)] = data
				mu.Unlock()
				timer.Reset(time.Second * 5)
				do = true
			case <-timer.C:
				if do {
					mu.Lock()
					for k, v := range work {
						_, err := v.w.Add(v.file)
						if err != nil {
							slog.Error(v.provider, "gitadd", err)
							continue
						}

						_, err = v.w.Commit("elephant", &git.CommitOptions{})
						if err != nil {
							slog.Error(v.provider, "commit", err)
							continue
						}

						err = v.r.Push(&git.PushOptions{})
						if err != nil {
							slog.Error(v.provider, "push", err)
							continue
						}

						delete(work, k)
					}
					mu.Unlock()

					do = false
				}
			}
		}
	}()
}

// TODO: this needs better commit messages somehow...
func GitPush(provider, file string, w *git.Worktree, r *git.Repository) {
	gitMu.Lock()
	defer gitMu.Unlock()

	pushChan <- PushData{
		provider: provider,
		file:     file,
		w:        w,
		r:        r,
	}
}
