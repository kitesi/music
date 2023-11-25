package play

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/shlex"
	stringUtils "github.com/kitesi/music/string-utils"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const maxSongsShown int = 20

func clearScreenDown() {
	fmt.Print("\x1b[0J")
}

func clearScreenUp() {
	fmt.Print("\x1b[1J")
}

func moveCursorUp(amount int) {
	fmt.Printf("\033[%dA", amount)
}

func moveCursorVerticalAbsolute(amount int) {
	fmt.Printf("\033[%dH", amount)
}

func moveCursorHorizontalAbsolute(amount int) {
	fmt.Printf("\033[%dG", amount)
}

func truncateString(val string, maxLength int) string {
	if len(val) > maxLength {
		return val[0:maxLength]
	}

	return val
}

func writeToScreen(query string, songs []string, musicPath string) error {
	clearScreenDown()
	fmt.Print("\r")

	terminalColumnSize, terminalRowSize, err := term.GetSize(int(os.Stdin.Fd()))

	if err != nil {
		return err
	}

	rowLimit := maxSongsShown

	// -3 for the shell prompt, query message, and horizontal line
	if terminalRowSize-3 < rowLimit {
		rowLimit = terminalRowSize - 3
	}

	var shownSongs []string

	if len(songs) > rowLimit {
		shownSongs = songs[:rowLimit]
	} else {
		shownSongs = songs
	}

	queryMessage := "Search: " + query
	horizontalLine := fmt.Sprintf("───────────────[%d]───────────────", len(songs))
	boilerPlate := fmt.Sprintf("%s\r\n%s\r\n", queryMessage, horizontalLine)
	linesFromSongs := 0

	fmt.Print(boilerPlate)

	for i, s := range shownSongs {
		fmt.Print(truncateString("- "+stringUtils.GetBareSongName(s, musicPath), terminalColumnSize))

		if i != len(shownSongs)-1 {
			fmt.Print("\r\n")
			linesFromSongs++
		}
	}

	linesFromBoilerPlate := strings.Count(boilerPlate, "\n") + (len(queryMessage) / terminalColumnSize) + (len(horizontalLine) / terminalColumnSize)

	// +1 so the cursor isn't on the last letter
	moveCursorHorizontalAbsolute(len(queryMessage) + 1)
	// +2 because of the horizontal line and new lines
	moveCursorUp(linesFromSongs + linesFromBoilerPlate)

	return nil
}

func liveQueryResults() error {
	subPlayCmd, subPlayArgs := generateCommand()
	subPlayTerms := []string{}

	subPlayCmd.Run = func(_ *cobra.Command, terms []string) {
		subPlayTerms = terms
	}

	subPlayCmd.SilenceErrors = true
	subPlayCmd.SilenceUsage = true
	subPlayCmd.SetFlagErrorFunc(func(_ *cobra.Command, _ error) error {
		return nil
	})

	oldState, err := term.MakeRaw(int(os.Stdin.Fd()))

	if err != nil {
		return err
	}

	defer term.Restore(int(os.Stdin.Fd()), oldState)

	lastQuery := ""
	lastSongs := []string{}
	query := ""

	clearScreenUp()
	moveCursorVerticalAbsolute(0)

InfiniteLoop:
	for {
		writeToScreen(query, lastSongs, subPlayArgs.musicPath)

		b := make([]byte, 1)
		_, err = os.Stdin.Read(b)

		if err != nil {
			return err
		}

		key := string(b[0])

		// shlex can't handle a starting quote
		if key == "\"" || key == "'" {
			continue
		}

		// todo: prob better way
		switch string(key) {
		// ctrl-c
		case "\x03":
			fmt.Print("\r")
			clearScreenDown()
			break InfiniteLoop
		// backspace
		case "\x7F":
			if query != "" {
				query = query[:len(query)-1]
			}
		// ctrl-u
		case "\x15":
			query = ""
		// ctrl-w
		case "\x17":
			tokens, err := shlex.Split(strings.TrimSpace(query))

			if err != nil || len(tokens) < 1 {
				break
			}

			query = strings.Join(tokens[0:len(tokens)-1], " ")
		case "\r":
			fmt.Print("\r")
			clearScreenDown()

			if len(lastSongs) == 0 {
				fmt.Println("No songs selected\r")
				return nil
			}

			for _, s := range lastSongs {
				fmt.Printf("- %s\r\n", stringUtils.GetBareSongName(s, subPlayArgs.musicPath))
			}

			runVLC(subPlayArgs, lastSongs)
			return nil
		}

		asciiCode := int(b[0])

		if asciiCode < 32 || asciiCode > 126 {
			if query == lastQuery {
				continue
			}
		} else {
			query += key
		}

		argsFromQuery, err := shlex.Split(query)

		if err != nil {
			continue
		}

		/*
		   seems like it might be slow tbh but the alternatives aren't great.

		   - Creating a new command each time in the for loop
		   - running a function that resets all the values of args to the defaults:
		   error prone and manual
		*/
		subPlayCmd.ResetFlags()
		addFlags(subPlayCmd, subPlayArgs)

		subPlayCmd.SetArgs(argsFromQuery)
		err = subPlayCmd.Execute()

		if err != nil {
			continue
		}

		// live parsing of music-path is just not efficient
		subPlayArgs.musicPath = ""
		setDefaultMusicPath(subPlayArgs)

		songs, err := getSongs(subPlayArgs, subPlayTerms)

		if err != nil {
			lastQuery = query
			continue
		}

		lastSongs = songs
		lastQuery = query
	}

	return nil
}