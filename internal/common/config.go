// Package common provides common functions used by all providers.
package common

import (
	"log/slog"
	"os"

	"github.com/knadh/koanf/parsers/toml/v2"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/providers/structs"
	"github.com/knadh/koanf/v2"
)

type Config struct {
	Icon     string `koanf:"icon" desc:"icon for provider" default:"depends on provider"`
	MinScore int32  `koanf:"min_score" desc:"minimum score for items to be displayed" default:"depends on provider"`
}

type ElephantConfig struct {
	ArgumentDelimiter      string `koanf:"argument_delimited" desc:"global delimiter for arguments" default:"#"`
	AutoDetectLaunchPrefix bool   `koanf:"auto_detect_launch_prefix" desc:"automatically detects uwsm, app2unit or systemd-run" default:"true"`
}

var elephantConfig ElephantConfig

func LoadGlobalConfig() {
	elephantConfig = ElephantConfig{
		ArgumentDelimiter:      "#",
		AutoDetectLaunchPrefix: true,
	}

	LoadConfig("elephant", elephantConfig)
}

func GetElephantConfig() *ElephantConfig {
	return &elephantConfig
}

func LoadConfig(provider string, config any) {
	defaults := koanf.New(".")

	err := defaults.Load(structs.Provider(config, "koanf"), nil)
	if err != nil {
		slog.Error(provider, "config", err)
		os.Exit(1)
	}

	userConfig := ProviderConfig(provider)

	if FileExists(userConfig) {
		user := koanf.New("")

		err := user.Load(file.Provider(userConfig), toml.Parser())
		if err != nil {
			slog.Error(provider, "config", err)
			os.Exit(1)
		}

		err = defaults.Merge(user)
		if err != nil {
			slog.Error(provider, "config", err)
			os.Exit(1)
		}

		err = defaults.Unmarshal("", &config)
		if err != nil {
			slog.Error(provider, "config", err)
			os.Exit(1)
		}
	} else {
		slog.Info(provider, "config", "not found. using default config")
	}
}
