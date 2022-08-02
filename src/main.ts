#!/usr/bin/env node
import { statSync, readdirSync } from 'fs';
import { exec as realExec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import yargs from 'yargs';

import { config } from './config.js';
import { doesSongPass } from './does-song-pass.js';
import * as playMusic from './play-music.js';
import { editSongInstallName, editSongList } from './pipe-through-editor.js';

import type { PlayMusicArgs } from './play-music.js';

let songsPath = config.get('path') as string;
let vlcPath = config.get('pathToVLC') as string;
let persist = config.get('persist') as boolean;
let sortType = config.get('sortType') as 'atimeMs' | 'ctimeMs' | 'mtimeMs';

const promiseBasedExec = promisify(realExec);
let exec = promiseBasedExec;
let timeoutTillExit = 100;

function getSongsByTerms(terms: string[], limit?: number, skip?: number) {
    const chosenSongs: string[] = [];
    let skipped = 0;

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

function sortByNew(a: string, b: string) {
    const songAStats = statSync(path.join(songsPath, a));
    const songBStats = statSync(path.join(songsPath, b));

    return songBStats[sortType] - songAStats[sortType];
}

async function getSongs(args: PlayMusicArgs) {
    let songs = getSongsByTerms(
        args.terms || [],
        // only give a limit if there is no need for sorting
        !args.new && !args['delete-old-first'] && !args['play-new-first']
            ? args.limit
            : undefined,
        args.skip
    );

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

function writeToScreen(query: string, msg: string) {
    process.stdout.write('\r');
    process.stdout.clearScreenDown();

    const queryMessage = 'Search: ' + query;

    process.stdout.write(
        queryMessage + '\n-----------------------\n' + msg + '\n'
    );

    process.stdout.moveCursor(
        queryMessage.length,
        -(msg.split('\n').length + 2)
    );
}

async function liveQueryResults() {
    const { stdin } = process;
    const wordsRegex = /[\s+#,]/g;

    stdin.setRawMode(true);
    stdin.resume();
    stdin.setEncoding('utf8');

    let query = '';
    let lastSongs: string[] = [];
    // @ts-expect-error
    let lastArgsFromQuery: PlayMusicArgs = {};

    writeToScreen('', '');

    const parser = yargs()
        // idk the typing is off for yargs
        // @ts-expect-error
        .command({
            command: '$0 [terms..]',
            describe: '',
            builder: playMusic.builder,
            handler: () => {},
        })
        .help(false)
        .strict(true);

    stdin.on('data', async function (key: string) {
        let prevQuery = query;

        switch (key) {
            // ctrl-c
            case '\u0003':
                process.stdout.write('\r');
                process.stdout.clearScreenDown();

                process.exit();
            // backspace
            case '\x7F':
                query = query.slice(0, query.length - 1);
                break;
            // ctrl-u
            case '\x15':
                query = '';
                break;
            // ctrl-w
            case '\x17':
                let words = query.trimEnd().split(wordsRegex);
                const seperators = query.match(wordsRegex);

                words = words.slice(0, words.length - 1);
                query = '';

                for (let i = 0; i < words.length; i++) {
                    query += words[i];

                    if (i != words.length - 1 && seperators && seperators[i]) {
                        query += seperators[i];
                    }
                }
                break;
            case '\r':
                process.stdout.write('\r');
                process.stdout.clearScreenDown();

                if (lastSongs.length === 0) {
                    console.log('No songs selected.');
                    process.exit(0);
                }

                playMusic.run({
                    args: lastArgsFromQuery,
                    exec,
                    songs: lastSongs,
                    songsPath,
                    vlcPath,
                });

                playMusic.message(lastSongs);

                await new Promise((res) =>
                    setTimeout(() => process.exit(), timeoutTillExit)
                );
                break;
        }

        const asciiCode = key.charCodeAt(0);

        if (asciiCode < 32 || asciiCode > 126) {
            if (query === prevQuery) {
                return;
            }
        } else {
            query += key;
        }

        let hasError = false;

        const argsFromQuery = parser
            .fail((msg: string) => {
                hasError = true;
                writeToScreen(query, msg);
                lastSongs = [];
            })
            .parse(query) as PlayMusicArgs;

        if (hasError) {
            return;
        }

        const songs = await getSongs(argsFromQuery);
        const msg = songs.slice(0, 20).join('\n');

        writeToScreen(query, msg);

        lastSongs = songs;
        lastArgsFromQuery = argsFromQuery;
    });

    return new Promise<void>((res) => {
        stdin.on('end', () => res());
    });
}

async function playMusicHandler(args: PlayMusicArgs) {
    if (
        args.terms.length === 0 &&
        !args.limit &&
        !args['dry-paths'] &&
        !args['play-new-first'] &&
        !args.new &&
        !args.live &&
        !args.editor
    ) {
        exec(`${vlcPath} --recursive=expand "${songsPath}"`);

        if (persist) {
            return;
        }

        return setTimeout(() => process.exit(0), timeoutTillExit);
    }

    if (args.live) {
        await liveQueryResults();
        process.exit();
    }

    const songs = await getSongs(args);

    if (songs.length === 0) {
        return console.error("Didn't match anything");
    }

    if (args['dry-paths']) {
        return console.log(
            songs.map((s) => path.join(songsPath, s)).join('\n')
        );
    }

    if (!args.limit && args.terms.length === 0 && !args.editor) {
        console.log('Playing all songs');
    } else {
        playMusic.message(songs);
    }

    playMusic.run({ args, exec, songs, songsPath, vlcPath });

    if (!persist) {
        setTimeout(() => process.exit(0), timeoutTillExit);
    }
}

yargs(process.argv.slice(2))
    .command({
        command: ['play [terms..]', 'p'],
        describe: 'play music',
        builder: playMusic.builder,
        // @ts-ignore
        handler: playMusicHandler,
    })
    .command({
        command: 'get-config-path',
        describe: 'get the config path',
        handler: () => {
            console.log(config.path);
        },
    })
    .command({
        command: ['install <id> <folder>', 'i'],
        describe: 'install music from youtube id or url',
        builder: (y) => {
            return y
                .option('format', {
                    type: 'string',
                    default: 'm4a',
                    alias: 'f',
                })
                .option('ytdl-args', {
                    type: 'string',
                    default: '',
                    alias: 'y',
                })
                .option('name', {
                    type: 'string',
                    alias: 'n',
                })
                .option('editor', {
                    type: 'boolean',
                    alias: 'e',
                })
                .conflicts('editor', 'name');
        },
        // @ts-expect-error
        handler: async ({
            folder,
            id,
            format,
            'ytdl-args': ytdlArgs,
            name: fileName,
            editor,
        }: {
            folder: string;
            id: string;
            format?: string;
            'ytdl-args'?: string;
            name?: string;
            editor?: boolean;
        }) => {
            const possibleFolders = readdirSync(songsPath);
            const adjustedFolder = folder.toLowerCase().replace(/\s+/g, '-');
            let selectedFolder = '';

            for (const possibleFolder of possibleFolders) {
                if (
                    possibleFolder.toLowerCase().replace(/\s+/g, '-') ===
                    adjustedFolder
                ) {
                    selectedFolder = possibleFolder;
                    break;
                }
            }

            if (!selectedFolder) {
                return console.error(`Invalid folder: ${folder}`);
            }

            const youtubeURL = id.startsWith('https://')
                ? id
                : `https://www.youtube.com/watch?v=${id}`;

            let outputTemplate = fileName
                ? fileName + '.%(ext)s'
                : '%(title)s.%(ext)s';

            if (editor) {
                outputTemplate =
                    (await editSongInstallName(youtubeURL)) + '.%(ext)s';
            }

            const child = exec(
                `youtube-dl -f ${format} -o "${path.join(
                    songsPath,
                    selectedFolder,
                    outputTemplate
                )}" ${ytdlArgs} -- "${youtubeURL}"`
            ).child;

            if (child.stdout) {
                child.stdout.on('data', (data) => console.log('' + data));
            }

            if (child.stderr) {
                child.stderr.on('data', (data) => console.log('' + data));
            }
        },
    })
    .alias('h', 'help')
    // @ts-expect-error
    .middleware((args: PlayMusicArgs) => {
        if (args['dry-run']) {
            // @ts-expect-error
            exec = () => {
                const promise = Object.assign(new Promise((res) => res), {
                    child: {
                        stdout: () => {},
                        stderr: () => {},
                    },
                });

                return promise;
            };

            timeoutTillExit = 0;
        }

        if (typeof args.persist !== 'undefined') {
            persist = args.persist;
        }

        if (args['vlc-path']) {
            vlcPath = args['vlc-path'];
        }

        if (args['songs-path']) {
            songsPath = path.resolve(args['songs-path']);
        }

        if (args['sort-type']) {
            // @ts-expect-error
            sortType = args['sort-type'] + 'timeMs';
        }
    })
    .strict().argv;
