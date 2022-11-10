package play

import (
	"strings"

	arrayUtils "github.com/kitesi/music/array-utils"
	"github.com/kitesi/music/commands/tags"
)

func doesSongPass(args *PlayArgs, savedTags []tags.Tag, terms []string, songPath string) bool {
	if len(terms) == 0 && len(args.tags) == 0 {
		return true
	}

	passedOneTerm := len(terms) == 0
	passedTagRequirement := len(args.tags) == 0

	songPath = strings.Replace(songPath, strings.ToLower(args.musicPath)+"/", "", 1)

	var validateTerm = func(term string) bool {
		return strings.Contains(songPath, term)
	}

	var validateTag = func(tag string) bool {
		return arrayUtils.Some(savedTags, func(savedTag tags.Tag) bool {
			return strings.Contains(savedTag.Name, tag) && arrayUtils.Some(savedTag.Songs, func(s string) bool {
				return s == songPath
			})
		})
	}

	for _, term := range terms {
		if validateQuery(term, validateTerm) {
			if strings.HasPrefix(term, "!") {
				return false
			}

			passedOneTerm = true
		}
	}

	for _, tag := range args.tags {
		if validateQuery(tag, validateTag) {
			if strings.HasPrefix(tag, "!") {
				return false
			}

			passedTagRequirement = true
		}
	}

	if !passedOneTerm && arrayUtils.Every(terms, func(term string) bool {
		return strings.HasPrefix(term, "!")
	}) {
		passedOneTerm = true
	}

	return passedOneTerm && passedTagRequirement
}

func validateQuery(query string, validator func(string) bool) bool {
	query = strings.TrimPrefix(strings.ToLower(query), "!")
	requiredSections := strings.Split(query, "#")

	return arrayUtils.Every(requiredSections, func(section string) bool {
		return arrayUtils.Some(strings.Split(section, ","), func(word string) bool {
			return validator(word)
		})
	})
}
