import path from 'path';
import { statSync, readdirSync } from 'fs';

import { editSongList } from './pipe-through-editor.js';
import { getTags } from './tags.js';

import type { PlayMusicArgs } from './play-music.js';

function validateQuery(query: string, validate: (word: string) => boolean) {
    let term = query.toLowerCase();

    const isExclusion = term.startsWith('!');

    if (isExclusion) {
        term = term.slice(1);
    }

    const requiredSections = term.split(/#\s*/);
    return requiredSections.every((section) =>
        section.split(/,\s*/).some((word) => validate(word))
    );
}

function doesSongPass(
    terms: string[],
    tags: string[] = [],
    songPath: string
): boolean {
    if (terms.length === 0 && tags.length === 0) {
        return true;
    }

    let passedOneTerm = false;
    let passedTagRequirement = tags.length === 0;

    for (const term of terms) {
        if (validateQuery(term, (w) => songPath.includes(w))) {
            const isExclusion = term.startsWith('!');

            if (isExclusion) {
                return false;
            }

            passedOneTerm = true;
        }
    }

    if (tags.length > 0) {
        const savedTags = getTags();

        for (let tag of tags) {
            if (
                validateQuery(tag, (w) =>
                    savedTags.some((t) => {
                        return (
                            t.name.includes(tag) &&
                            t.songs.includes(songPath.slice(1))
                        );
                    })
                )
            ) {
                const isExclusion = tag.startsWith('!');

                if (isExclusion) {
                    return false;
                }

                passedTagRequirement = true;
            }
        }
    }

    // this is to help in the cases where terms.length = 0 or when the terms are all exclusions
    // if they are all exclusions, they have passed every exclusion and as such, pass
    if (!passedOneTerm && terms.every((t) => t.startsWith('!'))) {
        passedOneTerm = true;
    }

    return passedOneTerm && passedTagRequirement;
}

function getSongsByTerms(args: PlayMusicArgs) {
    let { songsPath, terms, limit, skip } = args;
    const chosenSongs: string[] = [];

    // only give a limit if there is no need for sorting
    if (args.new || args.deleteOldFirst || args.playNewFirst) {
        limit = undefined;
    }

    if (limit && skip) {
        limit += skip;
    }

    function walk(dir: string) {
        const files = readdirSync(dir);

        for (const file of files) {
            const nextPath = path.join(dir, file);
            const stats = statSync(nextPath);

            if (!stats.isDirectory()) {
                if (
                    doesSongPass(
                        terms,
                        args.tags,
                        nextPath
                            .toLowerCase()
                            .replace(songsPath.toLowerCase(), '')
                    )
                ) {
                    chosenSongs.push(
                        nextPath.replace(songsPath + path.sep, '')
                    );

                    // all the walk process-es that started will still run, idk
                    // how to early exit out of function from inner function
                    if (limit && chosenSongs.length === limit) {
                        return;
                    }
                }
            } else {
                walk(nextPath);
            }
        }
    }

    walk(songsPath);
    return chosenSongs;
}

export async function getSongs(args: PlayMusicArgs, songsPath: string) {
    const sortType = (args.sortType +
        'timeMs') as `${typeof args.sortType}timeMs`;

    function sortByNew(a: string, b: string) {
        const songAStats = statSync(path.join(songsPath, a));
        const songBStats = statSync(path.join(songsPath, b));

        return songBStats[sortType] - songAStats[sortType];
    }

    let songs = getSongsByTerms(args);

    if (songs.length === 0) {
        return songs;
    }

    if (args.new || args.deleteOldFirst) {
        songs.sort(sortByNew);
    }

    if (args.skip) {
        songs = songs.slice(args.skip);
    }

    if (args.limit && songs.length > args.limit) {
        songs.length = args.limit;
    }

    // !args.new && !args['delete-old-first'] to make sure we don't uselessly sort again
    if (args.playNewFirst && !args.new && !args.deleteOldFirst) {
        songs.sort(sortByNew);
    }

    if (args.editor) {
        return await editSongList(songs);
    }

    return songs;
}
