import path from 'path';
import { statSync, readdirSync } from 'fs';

import { doesSongPass } from './does-song-pass.js';
import { editSongList } from './pipe-through-editor.js';

import type { PlayMusicArgs } from './play-music.js';

function getSongsByTerms(args: PlayMusicArgs) {
    let { 'songs-path': songsPath, terms, skip, limit } = args;
    const chosenSongs: string[] = [];
    let skipped = 0;

    // only give a limit if there is no need for sorting
    if (args.new || args['delete-old-first'] || args['play-new-first']) {
        limit = undefined;
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
                        nextPath
                            .toLowerCase()
                            .replace(songsPath.toLowerCase(), '')
                    )
                ) {
                    if (skip && skipped < skip) {
                        skipped++;
                        continue;
                    }

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
    const sortType = (args['sort-type'] +
        'timeMs') as `${typeof args['sort-type']}timeMs`;

    function sortByNew(a: string, b: string) {
        const songAStats = statSync(path.join(songsPath, a));
        const songBStats = statSync(path.join(songsPath, b));

        return songBStats[sortType] - songAStats[sortType];
    }

    let songs = getSongsByTerms(args);

    if (songs.length === 0) {
        return songs;
    }

    if (args.new || args['delete-old-first']) {
        songs.sort(sortByNew);
    }

    if (args.limit && songs.length > args.limit) {
        songs.length = args.limit;
    }

    // !args.new && !args['delete-old-first'] to make sure we don't uselessly sort again
    if (args['play-new-first'] && !args.new && !args['delete-old-first']) {
        songs.sort(sortByNew);
    }

    if (args.editor) {
        return await editSongList(songs);
    }

    return songs;
}
