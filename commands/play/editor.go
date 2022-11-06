package play

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

func editSongList(songs []string) ([]string, error) {
	editor := os.Getenv("EDITOR")

	if editor == "" {
		log.Fatal("No EDITOR env variable found")
	}

	file, err := os.CreateTemp("", "music-playlist-*.txt")

	if err != nil {
		return []string{}, err
	}

	defer file.Close()

	file.WriteString(strings.Join(songs, "\n"))

	cmd := exec.Command(editor, file.Name())

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return []string{}, err
	}

	cont, err := os.ReadFile(file.Name())

	if err != nil {
		log.Fatal(err)
	}

	editedSongs := strings.Split(string(cont), "\n")
	filteredEditedSongs := []string{}

	for _, s := range editedSongs {
		if s != "" {
			filteredEditedSongs = append(filteredEditedSongs, s)
		}
	}

	return filteredEditedSongs, nil
}
