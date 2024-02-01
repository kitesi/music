package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
)

const (
	MIN_TRACK_LEN            = 30
	MIN_LISTEN_TIME          = 4 * 60
	DEFAULT_INTERVAL_SECONDS = 10
	DEBUG                    = false
)

type LastfmConfig struct {
	Interval       int
	MinTrackLength int
	MinListenTime  int
}

type Config struct {
	MusicPath string
	Debug     bool
	LastFm    LastfmConfig
}

func GetConfigPath() (string, error) {
	configPath, err := os.UserConfigDir()

	if err != nil {
		return configPath, err
	}

	return path.Join(configPath, "go-music-kitesi", "config.json"), nil
}

func DefaultConfig() Config {
	musicPath, err := GetDefaultMusicPath()

	if err != nil {
		musicPath = ""
	}

	return Config{
		MusicPath: musicPath,
		Debug:     DEBUG,
		LastFm: LastfmConfig{
			Interval:       DEFAULT_INTERVAL_SECONDS,
			MinTrackLength: MIN_TRACK_LEN,
			MinListenTime:  MIN_LISTEN_TIME,
		},
	}
}

func GetConfig() (Config, error) {
	config := DefaultConfig()
	configPath, err := GetConfigPath()

	if err != nil {
		return config, errors.Wrap(err, "could not find config path")
	}

	f, err := os.Open(configPath)

	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return config, nil
		}

		return config, errors.Wrap(err, fmt.Sprintf("could not open config path (%s)", configPath))
	}

	defer f.Close()

	decoder := json.NewDecoder(f)
	err = decoder.Decode(&config)

	if err != nil {
		return config, errors.Wrap(err, "could not read config file")
	}

	return config, nil
}
