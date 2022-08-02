#!/usr/bin/env node
import { statSync, readdirSync } from 'fs';
import { exec as realExec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import yargs from 'yargs';

import { config } from './config.js';
import * as playMusic from './play-music.js';
import { editSongInstallName } from './pipe-through-editor.js';
import { getSongs } from './get-songs.js';
import { liveQueryResults } from './live-query-results.js';

import type { PlayMusicArgs } from './play-music.js';

const promiseBasedExec = promisify(realExec);
let exec = promiseBasedExec;
let timeoutTillExit = 100;

async function playMusicHandler(args: PlayMusicArgs) {
    const { 'vlc-path': vlcPath, 'songs-path': songsPath, persist } = args;

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
        await liveQueryResults(args, timeoutTillExit, exec);
        process.exit();
    }

    const songs = await getSongs(args, songsPath);

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
            const songsPath = config.get('path') as string;
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
    // @ts-ignore
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

        type SortTypeAlias = 'a' | 'c' | 'm';

        const valuesFromConfig = {
            persist: config.get('persist') as boolean,
            vlcPath: config.get('pathToVLC') as string,
            sortType: config.get('sortType') as `${SortTypeAlias}timeMs`,
            songsPath: config.get('path') as string,
        };

        if (!args['songs-path']) {
            args['songs-path'] = valuesFromConfig.songsPath;
        }

        if (!args['vlc-path']) {
            args['vlc-path'] = valuesFromConfig.vlcPath;
        }

        if (!('persist' in args)) {
            args.persist = valuesFromConfig.persist;
        }

        if (!args['sort-type']) {
            args['sort-type'] = valuesFromConfig.sortType.slice(
                0
            ) as SortTypeAlias;
        }
    })
    .strict().argv;
