package tags

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	arrayUtils "github.com/kitesi/music/array-utils"
	"github.com/kitesi/music/editor"
	stringUtils "github.com/kitesi/music/string-utils"
	"github.com/spf13/cobra"
)

type TagsCommandArgs struct {
	edit         bool
	check        bool
	shouldDelete bool
	musicPath    string
}

type Tag struct {
	Songs        []string `json:"songs"`
	CreationTime int64    `json:"creation_time"`
	ModifiedTime int64    `json:"modified_time"`
}

type Tags map[string]Tag

func GetTagJsonPath(musicPath string) string {
	return filepath.Join(musicPath, "playlists", "tags.json")
}

func GetPlaylistPath(musicPath string, playlistName string) string {
	return filepath.Join(musicPath, "playlists", playlistName+".m3u")
}

func GetStoredTags(musicPath string) (Tags, error) {
	storedTags := Tags{}

	content, err := os.ReadFile(GetTagJsonPath(musicPath))

	if err == nil {
		err = json.Unmarshal(content, &storedTags)

		if err != nil {
			return nil, err
		}
	} else if errors.Is(err, fs.ErrNotExist) {
		return storedTags, nil
	} else {
		return nil, err
	}

	return storedTags, nil
}

func Setup(rootCmd *cobra.Command) {
	args := TagsCommandArgs{}

	tagsCmd := &cobra.Command{
		Use:   "tags [tags..]",
		Short: "Manage tags",
		Long:  "Manage tags. Lists all the tags by default. If a tag is provided, this will list all the songs in that tag.",
		Run: func(cmd *cobra.Command, positional []string) {
			if err := tagsCommandRunner(&args, positional); err != nil {
				log.SetFlags(0)
				log.Fatal(err)
			}
		},
	}

	tagsCmd.Flags().BoolVarP(&args.edit, "editor", "e", false, "edit tags.json or a specific tag with $EDITOR")
	tagsCmd.Flags().BoolVarP(&args.check, "check", "c", false, "check if the songs exist under the given tags")
	tagsCmd.Flags().BoolVarP(&args.shouldDelete, "delete", "d", false, "delete a tag")
	tagsCmd.Flags().StringVarP(&args.musicPath, "music-path", "m", "", "the music path to use")

	rootCmd.AddCommand(tagsCmd)
}

func updateAPlaylistFile(musicPath string, playlistPath string, songs []string, playlistContent []string) error {
	for _, song := range songs {
		if song != "" {
			relativePath, err := filepath.Rel(filepath.Join(musicPath, "playlists"), song)

			if err != nil {
				return err
			}

			playlistContent = append(playlistContent, fmt.Sprintf("%s\n", relativePath))
		}
	}

	os.WriteFile(playlistPath, []byte(strings.Join(playlistContent, "")), 0666)
	return nil
}

func tagsCommandRunner(args *TagsCommandArgs, positional []string) error {
	if args.shouldDelete {
		if args.edit {
			return errors.New("can't have --delete and --editor together")
		}

		if len(positional) == 0 {
			return errors.New("can't use --delete without a tag")
		}
	}

	if args.check && args.edit {
		return errors.New("can't have --check and --editor together")
	}

	if args.musicPath == "" {
		defaultMusicPath, err := stringUtils.GetDefaultMusicPath()

		if err != nil {
			return err
		}

		args.musicPath = defaultMusicPath
	}

	if args.check {
		storedTags, err := GetStoredTags(args.musicPath)

		if err != nil {
			return err
		}

		for _, requestedTagName := range positional {
			tag, ok := storedTags[requestedTagName]

			if !ok {
				fmt.Fprintf(os.Stderr, "error: tag \"%s\" does not exist\n", requestedTagName)
				continue
			}

			allSongsExist := true

			for _, song := range tag.Songs {
				_, err := os.Stat(song)

				if errors.Is(err, fs.ErrNotExist) {
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

	// create the playlists directory if it doesn't exist
	_, err := os.Stat(filepath.Join(args.musicPath, "playlists"))

	if errors.Is(err, fs.ErrNotExist) {
		err = os.Mkdir(filepath.Join(args.musicPath, "playlists"), 0777)
	} else if err != nil {
		return err
	}

	// create the tags.json file if it doesn't exist
	_, err = os.Stat(GetTagJsonPath(args.musicPath))

	if errors.Is(err, fs.ErrNotExist) {
		err = os.WriteFile(GetTagJsonPath(args.musicPath), []byte("{}"), 0666)
	} else if err != nil {
		return err
	}

	if len(positional) == 0 {
		if args.edit {
			_, err := editor.EditFile(GetTagJsonPath(args.musicPath))

			if err != nil {
				return err
			}

			storedTags, err := GetStoredTags(args.musicPath)

			if err != nil {
				return err
			}

			files, err := os.ReadDir(filepath.Join(args.musicPath, "playlists"))

			if err != nil {
				return err
			}

			// remove the files for the deleted playlists
			for _, file := range files {
				if !strings.HasSuffix(file.Name(), ".m3u") {
					continue
				}

				tagName := strings.TrimSuffix(file.Name(), ".m3u")

				if _, ok := storedTags[tagName]; !ok {
					// don't exit because of it here because it's not a big deal
					if err := os.Remove(filepath.Join(args.musicPath, "playlists", file.Name())); err != nil {
						fmt.Printf("error [%s]: %s\n", file.Name(), err)
					}
				} else {
					err = updateAPlaylistFile(args.musicPath, GetPlaylistPath(args.musicPath, tagName), storedTags[tagName].Songs, []string{fmt.Sprintf("#EXTM3U\n#PLAYLIST:%s\n", tagName)})

					if err != nil {
						fmt.Printf("error [%s]: %s\n", file.Name(), err)
					}
				}

			}

			return nil
		}

		storedTags, err := GetStoredTags(args.musicPath)

		if err != nil {
			return err
		}

		for k := range storedTags {
			fmt.Println(k)
		}

		return nil
	}

	storedTags, err := GetStoredTags(args.musicPath)

	if err != nil {
		return err
	}

	for _, requestedTagName := range positional {
		tag, ok := storedTags[requestedTagName]

		if args.edit {
			content, err := editor.CreateAndModifyTemp("", requestedTagName+"-*.txt", strings.Join(tag.Songs, "\n"))

			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				continue
			}

			if err = ChangeSongsInTag(args.musicPath, requestedTagName, strings.Split(content, "\n"), false); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			continue
		}

		if !ok {
			fmt.Fprintf(os.Stderr, "Error: tag \"%s\" does not exist\n", requestedTagName)
			continue
		}

		if args.shouldDelete {
			delete(storedTags, requestedTagName)

			if err := updateTagsJsonFile(&storedTags, args.musicPath); err != nil {
				fmt.Fprintln(os.Stderr, err)
			}

			continue
		}

		fmt.Printf("Name: %s, Amount: %d, Creation: %s, Modified: %s\n", requestedTagName, len(tag.Songs), formatTime(tag.ModifiedTime), formatTime(tag.CreationTime))
		fmt.Println(strings.Join(tag.Songs, "\n"))

	}
	return nil
}

func formatTime(timeStamp int64) string {
	return time.Unix(timeStamp, 0).Format("2006-01-02 15:04:05")
}

func ChangeSongsInTag(musicPath string, tagName string, songs []string, shouldAppend bool) error {
	storedTags, err := GetStoredTags(musicPath)

	if err != nil {
		return err
	}

	tag, ok := storedTags[tagName]
	now := time.Now().Unix()
	playlistPath := GetPlaylistPath(musicPath, tagName)
	playlistContent := []string{}

	if !ok {
		tag = Tag{CreationTime: now, ModifiedTime: now, Songs: arrayUtils.FilterEmptyStrings(songs)}
	} else if shouldAppend {
		for _, song := range songs {
			if song != "" && !arrayUtils.Includes(tag.Songs, song) {
				tag.Songs = append(tag.Songs, song)
			}
		}
	} else {
		tag.Songs = arrayUtils.FilterEmptyStrings(songs)
	}

	if shouldAppend {
		_, err := os.Stat(playlistPath)

		if errors.Is(err, fs.ErrNotExist) {
			playlistContent = []string{fmt.Sprintf("#EXTM3U\n#PLAYLIST:%s\n", tagName)}
		} else if err != nil {
			return err
		} else {
			c, err := os.ReadFile(playlistPath)
			playlistContent = []string{string(c)}

			if err != nil {
				return err
			}
		}
	} else {
		playlistContent = []string{fmt.Sprintf("#EXTM3U\n#PLAYLIST:%s\n", tagName)}
	}

	err = updateAPlaylistFile(musicPath, playlistPath, tag.Songs, playlistContent)

	if err != nil {
		return err
	}

	tag.ModifiedTime = now
	storedTags[tagName] = tag

	return updateTagsJsonFile(&storedTags, musicPath)
}

func updateTagsJsonFile(tags *Tags, musicPath string) error {
	tagsString, err := json.Marshal(tags)

	if err != nil {
		return err
	}

	return os.WriteFile(GetTagJsonPath(musicPath), tagsString, 0666)
}
