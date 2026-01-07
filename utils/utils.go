package utils

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	LASTFM_CREDENTIALS_FILE  = ".lastfm-credentials"
	SPOTIFY_CREDENTIALS_FILE = ".spotify-credentials"
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
	if !strings.HasSuffix(musicPath, "/") {
		musicPath += "/"
	}

	return strings.Replace(song, musicPath, "", 1)
}

func CreateAndModifyTemp(dir, pattern, preloadedContent string) (string, error) {
	if os.Getenv("EDITOR") == "" {
		return "", errors.New("$EDITOR is not set")
	}

	file, err := os.CreateTemp("", pattern)

	if err != nil {
		return "", err
	}

	defer os.Remove(file.Name())
	defer file.Close()

	if preloadedContent != "" {
		file.WriteString(preloadedContent)
	}

	newContent, err := EditFile(file.Name())

	if err != nil {
		return "", err
	}

	return string(newContent), nil

}

func EditFile(fileName string) ([]byte, error) {
	editor := os.Getenv("EDITOR")

	if editor == "" {
		return nil, errors.New("$EDITOR is not set")
	}

	cmd := exec.Command(editor, fileName)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	content, err := os.ReadFile(fileName)

	if err != nil {
		return nil, err
	}

	return content, err
}

func Every[T any](arr []T, validator func(T) bool) bool {
	for _, el := range arr {
		if !validator(el) {
			return false
		}
	}

	return true
}

func Some[T any](arr []T, validator func(T) bool) bool {
	for _, el := range arr {
		if validator(el) {
			return true
		}
	}

	return false
}

func Includes[T comparable](arr []T, item T) bool {
	return Some(arr, func(i T) bool {
		return i == item
	})
}

func FilterEmptyStrings(arr []string) []string {
	output := make([]string, 0, len(arr))

	for _, item := range arr {
		if item != "" {
			output = append(output, item)
		}
	}

	return output
}
