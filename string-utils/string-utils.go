package stringUtils

import (
	"os"
	"path/filepath"
	"strings"
)

func GetDefaultMusicPath() (string, error) {
	envMusicPath, _ := os.LookupEnv("MUSIC_PATH")

	if envMusicPath != "" {
		return envMusicPath, nil
	}

	dirname, err := os.UserHomeDir()

	if err != nil {
		return "", err
	}

	return filepath.Join(dirname, "Music"), err
}

func GetBareSongName(song string, musicPath string) string {
	return strings.Replace(song, musicPath+"/", "", 1)
}
