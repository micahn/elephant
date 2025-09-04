package main

import (
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"syscall"

	"github.com/abenz1267/elephant/internal/common"
)

func Activate(qid uint32, identifier, action string, arguments string) {
	toRun := ""
	prefix := common.LaunchPrefix(config.LaunchPrefix)

	splits := strings.Split(arguments, common.GetElephantConfig().ArgumentDelimiter)
	if len(splits) > 1 {
		arguments = splits[1]
	} else {
		arguments = ""
	}

	parts := strings.Split(identifier, ":")

	if len(parts) == 2 {
		for _, v := range files[parts[0]].Actions {
			if v.Action == parts[1] {
				toRun = v.Exec
				break
			}
		}
	} else {
		toRun = files[parts[0]].Exec
	}

	if files[parts[0]].Terminal {
		toRun = common.WrapWithTerminal(toRun)
	}

	cmd := exec.Command("sh", "-c", strings.TrimSpace(fmt.Sprintf("%s %s %s", prefix, toRun, arguments)))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setsid: true,
	}

	err := cmd.Start()
	if err != nil {
		slog.Error(Name, "activate", identifier, "error", err)
		return
	} else {
		go func() {
			cmd.Wait()
		}()
	}

	if config.History {
		var last uint32

		for k := range results.Queries[qid] {
			if k > last {
				last = k
			}
		}

		if last != 0 {
			h.Save(results.Queries[qid][last], identifier)
		} else {
			h.Save("", identifier)
		}
	}

	slog.Info(Name, "activated", identifier)
}
