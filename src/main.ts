#!/usr/bin/env node
import { readdirSync } from 'fs';
import { exec as realExec } from 'child_process';
import { promisify } from 'util';
import path from 'path';
import yargs from 'yargs';

import * as playMusic from './play-music.js';
import { editSongInstallName } from './pipe-through-editor.js';
import { getSongs } from './get-songs.js';
import { liveQueryResults } from './live-query-results.js';

import type { PlayMusicArgs } from './play-music.js';
import { changeSongsInTag } from './tags.js';
import { songsPath as defaultSongsPath } from './get-default-songs-path.js';

const promiseBasedExec = promisify(realExec);
let exec = promiseBasedExec;
let timeoutTillExit = 100;

async function playMusicHandler(args: PlayMusicArgs) {
    const { vlcPath, songsPath, persist } = args;

    if (
        args.terms.length === 0 &&
        !args.limit &&
        !args.dryPaths &&
        !args.playNewFirst &&
        !args.new &&
        !args.live &&
        !args.editor &&
        (!args.tags || args.tags.length === 0)
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

    if (args.addToTag) {
        changeSongsInTag(
            songsPath,
            args.addToTag,
            songs.map((s) => s.toLowerCase()),
            true
        );
    }

    if (args.setToTag) {
        changeSongsInTag(
            songsPath,
            args.setToTag,
            songs.map((s) => s.toLowerCase()),
            false
        );
    }

    if (args.dryPaths) {
        return console.log(
            songs.map((s) => path.join(songsPath, s)).join('\n')
        );
    }

    if (
        !args.limit &&
        args.terms.length === 0 &&
        !args.editor &&
        (!args.tags || args.tags.length === 0)
    ) {
        console.log('Playing all songs');
    } else {
        playMusic.message(songs);
    }

    playMusic.execVLC({ args, exec, songs, songsPath, vlcPath });

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
                .option('songs-path', {
                    type: 'string',
                    default: defaultSongsPath,
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
            songsPath,
        }: {
            folder: string;
            id: string;
            format?: string;
            'ytdl-args'?: string;
            name?: string;
            editor?: boolean;
            songsPath: string;
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
    // @ts-ignore
    .middleware((args: PlayMusicArgs) => {
        if (args.dryRun) {
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
    })
    .strict().argv;
