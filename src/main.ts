#!/usr/bin/env node

import { statSync, readdirSync } from 'fs';
import { exec as realExec } from 'child_process';
import { promisify } from 'util';
import { config } from './config.js';
import path from 'path';
import yargs from 'yargs';
import chalk from 'chalk';

let songsPath = config.get('path') as string;
let vlcPath = config.get('pathToVLC') as string;
let persist = config.get('persist') as boolean;
let sortType = config.get('sortType') as 'atimeMs' | 'ctimeMs' | 'mtimeMs';

function logErrors(reason: any) {
    console.error('Error: ' + (reason?.message || `\n\n${reason}`));
}

const promiseBasedExec = promisify(realExec);
let isDryRun = false;
let exec = promiseBasedExec;

const timeoutTillExit = isDryRun ? 0 : 1200;

function doesSongPass(terms: string[], songPath: string): boolean {
    if (terms.length === 0) {
        return true;
    }

    let passedOneTerm = false;

    for (let term of terms) {
        term = term.toLowerCase();

        const isExclusion = term.startsWith('!');

        if (isExclusion) {
            term = term.slice(1);
        }

        const requiredSections = term.split(/#\s*/);

        if (
            requiredSections.every((s) =>
                s.split(/,\s*/).some((w) => songPath.includes(w))
            )
        ) {
            if (isExclusion) {
                return false;
            }

            passedOneTerm = true;
        }
    }

    if (terms.every((t) => t.startsWith('!'))) {
        return true;
    }

    return passedOneTerm;
}

function getSongsByTerms(terms: string[], limit?: number) {
    const chosenSongs: string[] = [];

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
                    // all the walk process-es that started will still run, idk
                    // how to early exit out of function from inner function
                    if (limit && chosenSongs.length === limit) {
                        return;
                    } else {
                        chosenSongs.push(
                            nextPath.replace(songsPath + path.sep, '')
                        );
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

// const line = '─'.repeat(60);

interface DefaultCommandArgs {
    terms?: string[];
    limit: number;
    new: boolean;
    persist?: boolean;
    'dry-run': boolean;
    'dry-paths': boolean;
    'play-new-first': boolean;
    'delete-old-first': boolean;
    'vlc-path': string;
    'songs-path': string;
    'sort-type': 'a' | 'c' | 'm';
}

async function defaultCommandHandler(args: DefaultCommandArgs) {
    if (
        (!args.terms || args.terms.length === 0) &&
        !args.limit &&
        !args['dry-paths'] &&
        !args['play-new-first'] &&
        !args.new
    ) {
        console.log('Playing all songs');
        exec(`${vlcPath} --recursive=expand "${songsPath}"`);

        if (persist) {
            return;
        }

        return setTimeout(() => process.exit(0), timeoutTillExit);
    }

    let songs = getSongsByTerms(
        args.terms || [],
        // only give a limit if there is no need for sorting
        !args.new && !args['delete-old-first'] && !args['play-new-first']
            ? args.limit
            : undefined
    );

    if (songs.length === 0) {
        return console.error("Didn't match anything");
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

    if (args['dry-paths']) {
        return console.log(
            songs.map((s) => path.join(songsPath, s)).join('\n')
        );
    }

    if (!args.limit && (!args.terms || args.terms.length === 0)) {
        console.log('Playing all songs');
    } else {
        const playingMessage = `Playing: [${songs.length}]`;

        console.log(
            `${playingMessage}\n` +
                songs.map((e) => chalk.redBright('- ' + e)).join('\n')
        );
    }

    exec(
        `${vlcPath} ${songs
            .map(
                (s) =>
                    `"${songsPath}/${s}" ${
                        args.new || args['play-new-first'] ? '--no-random' : ''
                    }`
            )
            .join(' ')}`
    ).catch(logErrors);

    if (!persist) {
        setTimeout(() => process.exit(0), timeoutTillExit);
    }
}

yargs(process.argv.slice(2))
    .command({
        command: ['play [terms..]', 'p'],
        describe: 'play music',
        builder: (y) =>
            y
                .option('dry-run', {
                    alias: 'd',
                    type: 'boolean',
                })
                .option('limit', {
                    alias: 'l',
                    type: 'number',
                })
                .option('new', {
                    alias: 'n',
                    type: 'boolean',
                })
                .option('play-new-first', {
                    type: 'boolean',
                    alias: 'pnf',
                })
                .option('delete-old-first', {
                    type: 'boolean',
                    alias: 'dof',
                })
                .option('dry-paths', {
                    type: 'boolean',
                    alias: 'p',
                })
                .option('persist', {
                    type: 'boolean',
                })
                .option('vlc-path', {
                    type: 'string',
                })
                .option('sort-type', {
                    type: 'string',
                    choices: ['a', 'm', 'c'],
                })
                .option('songs-path', {
                    type: 'string',
                })
                .positional('terms', {
                    type: 'string',
                    array: true,
                }),
        // @ts-ignore
        handler: defaultCommandHandler,
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
                });
        },
        // @ts-expect-error
        handler: ({
            folder,
            id,
            format,
            'ytdl-args': ytdlArgs,
            name: fileName,
        }: {
            folder: string;
            id: string;
            format?: string;
            'ytdl-args'?: string;
            name?: string;
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

            const child = exec(
                `youtube-dl -f ${format} -o "${path.join(
                    songsPath,
                    selectedFolder,
                    (fileName && fileName + '.%(ext)s') || '%(title)s.%(ext)s'
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
    .middleware((args: DefaultCommandArgs) => {
        isDryRun = args['dry-run'];

        if (isDryRun) {
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
        }

        if (typeof args.persist !== 'undefined') {
            persist = args.persist;
        }

        if (args['vlc-path']) {
            vlcPath = args['vlc-path'];
        }

        if (args['songs-path']) {
            songsPath = path.join(process.cwd(), args['songs-path']);
        }

        if (args['sort-type']) {
            // @ts-expect-error
            sortType = args['sort-type'] + 'timeMs';
        }
    })
    .strict().argv;
