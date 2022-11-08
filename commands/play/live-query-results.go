package play

import (
	"fmt"
	"os"
	"strings"

	"github.com/google/shlex"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const maxSongsShoen int = 20

func clearScreenDown() {
	fmt.Print("\x1b[0J")
}

func moveCursorUp(amount int) {
	fmt.Printf("\033[%dA", amount)
}

func moveCursorAbsolute(amount int) {
	fmt.Printf("\033[%dG", amount)
}

func writeToScreen(query string, songs []string) error {
	fmt.Print("\r")
	clearScreenDown()

	var showenSongs []string

	if len(songs) > maxSongsShoen {
		showenSongs = songs[:maxSongsShoen]
	} else {
		showenSongs = songs
	}

	queryMessage := "Search: " + query
	msg := strings.Join(showenSongs, "\r\n")

	fmt.Print(queryMessage + "\r\n----------------------------\r\n" + msg + "\r\n")

	lines := strings.Count(msg, "\n")
	terminalColumnSize, _, err := term.GetSize(int(os.Stdin.Fd()))

	if err != nil {
		return err
	}

	for _, s := range showenSongs {
		if len(s) > terminalColumnSize {
			lines++
		}
	}

	moveCursorAbsolute(len(queryMessage) + 1)
	moveCursorUp(lines + 3)

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

	writeToScreen("", []string{})

	lastQuery := ""
	lastSongs := []string{}
	query := ""

InfiniteLoop:
	for {
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
				fmt.Println("No songs selected")
				return nil
			}

			vlcArgs := []string{}

			for _, s := range lastSongs {
				vlcArgs = append(vlcArgs, s)
				fmt.Printf("- %s\r\n", strings.Replace(s, subPlayArgs.musicPath+"/", "", 1))
			}

			runVLC(subPlayArgs, vlcArgs)
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

		songs, err := getSongs(subPlayArgs, subPlayTerms)

		if err != nil {
			lastQuery = query
			writeToScreen(query, lastSongs)
			continue
		}

		writeToScreen(query, songs)

		lastSongs = songs
		lastQuery = query
	}

	return nil
}
