package tags

import (
	// "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/kitesi/music/utils"
	"github.com/spf13/cobra"
)

type TagsCommandArgs struct {
	edit         bool
	check        bool
	shouldDelete bool
	debug        bool
	musicPath    string
}

func GetTagPath(musicPath string, tagName string) string {
	return filepath.Join(musicPath, "tags", tagName+".m3u")
}

func GetStoredTags(musicPath string) (map[string][]string, error) {
	storedTags := make(map[string][]string)
	tagsDirectory := filepath.Join(musicPath, "tags")

	files, err := os.ReadDir(filepath.Join(musicPath, "tags"))

	// if the tags directory doesn't exist, create the directory and return an empty map
	if os.IsNotExist(err) {
		err := os.Mkdir(tagsDirectory, 0777)
		if err != nil {
			return nil, fmt.Errorf("could not create tags directory: %w", err)
		}
		return storedTags, nil
	}

	if err != nil {
		return storedTags, nil
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".m3u") {
			songs := []string{}

			tagPath := filepath.Join(musicPath, "tags", file.Name())
			tagContent, err := os.ReadFile(tagPath)

			if err != nil {
				return nil, fmt.Errorf("could not read tag file \"%s\": %w", tagPath, err)
			}

			// sort of a naive implementation, and assumes the user won't modify the tag file
			for _, line := range strings.Split(string(tagContent), "\n") {
				if strings.HasPrefix(line, "#") || line == "" {
					continue
				}

				songPath := filepath.Join(musicPath, "tags", line)

				if filepath.IsAbs(line) {
					songPath = line
				}

				songs = append(songs, songPath)
			}

			storedTags[strings.TrimSuffix(file.Name(), ".m3u")] = songs
		}
	}

	return storedTags, nil
}

func Setup() *cobra.Command {
	args := TagsCommandArgs{}

	tagsCmd := &cobra.Command{
		Use:   "tags [tags..]",
		Short: "Manage tags",
		Long:  "Manage tags. Lists all the tags by default. If a tag is provided, this will list all the songs in that tag.",
		Run: func(cmd *cobra.Command, positional []string) {
			if err := tagsCommandRunner(&args, positional); err != nil {
				if args.debug {
					fmt.Fprintf(os.Stderr, "error: %+v\n", err)
				} else {
					fmt.Fprintf(os.Stderr, "error: %s\n", err)
				}
			}
		},
	}

	config, err := utils.GetConfig()

	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %+v\n", err)
	}

	tagsCmd.Flags().BoolVarP(&args.edit, "edit", "e", false, "edit tags.json or a specific tag with $EDITOR")
	tagsCmd.Flags().BoolVarP(&args.check, "check", "c", false, "check if the songs exist under the given tags")
	tagsCmd.Flags().BoolVarP(&args.shouldDelete, "delete", "d", false, "delete a tag")
	tagsCmd.Flags().BoolVar(&args.debug, "debug", config.Debug, "enable debug mode")
	tagsCmd.Flags().StringVarP(&args.musicPath, "music-path", "m", config.MusicPath, "the music path to use")

	return tagsCmd
}

func tagsCommandRunner(args *TagsCommandArgs, positional []string) error {
	if len(positional) == 0 && args.edit {
		return errors.New("can't use --edit without a tag")
	} else if len(positional) == 0 && args.shouldDelete {
		return errors.New("can't use --delete without a tag")
	} else if args.edit && args.shouldDelete {
		return errors.New("can't use --delete with --edit")
	} else if args.check && args.edit {
		return errors.New("can't use --edit with --check")
	} else if args.check && args.shouldDelete {
		return errors.New("can't use --delete with --check")
	}

	if args.check {
		storedTags, err := GetStoredTags(args.musicPath)

		if err != nil {
			return fmt.Errorf("could not get stored tags: %w", err)
		}

		for _, requestedTagName := range positional {
			tag, ok := storedTags[requestedTagName]

			if !ok {
				fmt.Fprintf(os.Stderr, "error: tag \"%s\" does not exist\n", requestedTagName)
				continue
			}

			allSongsExist := true

			for _, song := range tag {
				_, err := os.Stat(song)

				if os.IsNotExist(err) {
					allSongsExist = false
					fmt.Fprintf(os.Stderr, "error: song \"%s\" does not exist\n", song)
				} else if err != nil {
					return err
				}
			}

			if allSongsExist {
				fmt.Printf("all songs in tag \"%s\" exist\n", requestedTagName)
			}

		}

		return nil
	}

	if len(positional) == 0 {
		storedTags, err := GetStoredTags(args.musicPath)

		if err != nil {
			return fmt.Errorf("could not get stored tags: %w", err)
		}

		for k := range storedTags {
			fmt.Println(k)
		}

		return nil
	}

	_, err := os.Stat(filepath.Join(args.musicPath, "tags"))

	// create the tags directory if it doesn't exist
	if os.IsNotExist(err) {
		if args.shouldDelete {
			return errors.New("there are no tags to delete")
		}

		err = os.Mkdir(filepath.Join(args.musicPath, "tags"), 0777)
	} else if err != nil {
		return fmt.Errorf("could not get tags directory: %w", err)
	}

	storedTags, err := GetStoredTags(args.musicPath)

	if err != nil {
		return fmt.Errorf("could not get stored tags: %w", err)
	}

	for _, requestedTagName := range positional {
		tag, ok := storedTags[requestedTagName]

		if args.edit {
			tagPath := GetTagPath(args.musicPath, requestedTagName)

			if !ok {
				err := os.WriteFile(tagPath, []byte("#EXTM3U\n#PLAYLIST:"+requestedTagName+"\n"), 0666)

				if err != nil {
					return fmt.Errorf("could not write tag file: %w", err)
				}
			}
			_, err = utils.EditFile(tagPath)

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}
		} else if !ok {
			fmt.Fprintf(os.Stderr, "error: tag \"%s\" does not exist\n", requestedTagName)
		} else if args.shouldDelete {
			delete(storedTags, requestedTagName)

			if err := os.Remove(GetTagPath(args.musicPath, requestedTagName)); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}
		} else {
			fmt.Printf("Name: %s, Amount: %d\n", requestedTagName, len(tag))
			fmt.Println(strings.Join(tag, "\n"))
		}
	}

	return nil
}

func formatTime(timeStamp int64) string {
	return time.Unix(timeStamp, 0).Format("2006-01-02 15:04:05")
}

func ChangeSongsInTag(musicPath string, tagName string, songs []string, shouldAppend bool) error {
	storedTags, err := GetStoredTags(musicPath)

	if err != nil {
		return fmt.Errorf("could not get stored tags: %w", err)
	}

	tagPath := GetTagPath(musicPath, tagName)
	tagContent := []string{}
	tagSongs, ok := storedTags[tagName]

	if shouldAppend {
		_, err := os.Stat(tagPath)

		if os.IsNotExist(err) {
			tagContent = []string{fmt.Sprintf("#EXTM3U\n#PLAYLIST:%s", tagName)}
		} else if err != nil {
			return fmt.Errorf("could not get tag file: %w", err)
		} else {
			c, err := os.ReadFile(tagPath)

			if err != nil {
				return fmt.Errorf("could not read tag file: %w", err)
			}

			tagContent = []string{string(c)}
		}
	} else {
		tagContent = []string{fmt.Sprintf("#EXTM3U\n#PLAYLIST:%s", tagName)}
		tagSongs = []string{}
	}

	if !ok || tagSongs == nil {
		tagSongs = []string{}
	}

	for _, song := range songs {
		if song != "" && !utils.Includes(tagSongs, song) {
			relativePath, err := filepath.Rel(filepath.Join(musicPath, "tags"), song)

			if err != nil {
				fmt.Fprintf(os.Stderr, "error: could not get relative path for \"%s\": %s\n", song, err)
				continue
			}

			tagContent = append(tagContent, relativePath)
			tagSongs = append(tagSongs, song)
		}
	}

	err = os.WriteFile(tagPath, []byte(strings.Join(tagContent, "\n")), 0666)

	if err != nil {
		return fmt.Errorf("could not write tag file: %w", err)
	}

	storedTags[tagName] = tagSongs
	return nil
}
