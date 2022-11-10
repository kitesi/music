package editor

import (
	"errors"
	"os"
	"os/exec"
)

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
