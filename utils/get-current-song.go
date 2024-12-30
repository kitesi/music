package utils

import (
	"os/exec"
	"strings"
)

type SongMetadata struct {
	Artist string
	Track  string
	Length string
}

type SongMetadataError string

const (
	cantGetMetadata SongMetadataError = "playerctl - could not get metadata"
	missingFields   SongMetadataError = "playerctl - could get metadata but not the necessary fields"
)

func (e SongMetadataError) Error() string {
	return string(e)
}

func GetCurrentPlayingSong() (SongMetadata, error) {
	metadataCmd := exec.Command("playerctl", "-p", "vlc", "metadata")
	metadataOutput, err := metadataCmd.Output()

	if err != nil {
		return SongMetadata{}, cantGetMetadata
	}

	metadata := make(map[string]string)

	for _, line := range strings.Split(string(metadataOutput), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// split by whitespace
		sections := strings.Fields(line)

		if len(sections) < 3 {
			continue
		}

		_, key, value := sections[0], sections[1], strings.Join(sections[2:], " ")
		metadata[key] = value
	}

	if metadata["xesam:artist"] == "" || metadata["xesam:title"] == "" || metadata["vlc:length"] == "" {
		return SongMetadata{}, missingFields
	}

	return SongMetadata{
		Artist: metadata["xesam:artist"],
		Track:  metadata["xesam:title"],
		Length: metadata["vlc:length"],
	}, nil
}
