#!/usr/bin/env node

import { statSync, readdirSync } from 'fs';
import { exec as realExec } from 'child_process';
import { promisify } from 'util';
import { config } from './config.js';
import path from 'path';
import yargs from 'yargs';
import chalk from 'chalk';

const songsPath = config.get('path') as string;

function logErrors(reason: any) {
    console.error('Error: ' + (reason?.message || `\n\n${reason}`));
}

const isDryRun =
    process.argv.includes('--dry-run') || process.argv.includes('-d');

const promiseBasedExec = promisify(realExec);

const exec = isDryRun
    ? () => {
          // @ts-expect-error
          const promise: ReturnType<typeof promiseBasedExec> = Object.assign(
              new Promise((res) => res),
              {
                  child: {
                      stdout: () => {},
                      stderr: () => {},
                  },
              }
          );

          return promise;
      }
    : promiseBasedExec;

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

    return passedOneTerm;
}

function getSongsByTerms(terms: string[]) {
    const chosenSongs: string[] = [];

    function walk(dir: string) {
        const files = readdirSync(dir);

        for (const file of files) {
            const nextPath = path.join(dir, file);

            if (file.includes('.')) {
                if (
                    doesSongPass(
                        terms,
                        nextPath
                            .toLowerCase()
                            .replace(songsPath.toLowerCase(), '')
                    )
                ) {
                    chosenSongs.push(nextPath.replace(songsPath + '/', ''));
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

    return songBStats.ctimeMs - songAStats.ctimeMs;
}

// const line = '─'.repeat(60);

async function defaultCommandHandler(args: {
    terms?: string[];
    limit: number;
    new: boolean;
    'dry-run': boolean;
    'dry-paths': boolean;
    'play-new-first': boolean;
    'delete-old-first': boolean;
}) {
    if (
        (!args.terms || args.terms.length === 0) &&
        !args.limit &&
        !args['dry-paths']
    ) {
        console.log('Playing all songs');
        exec(`vlc --recursive=expand "${songsPath}"`);

        return setTimeout(() => process.exit(0), timeoutTillExit);
    }

    let songs = getSongsByTerms(args.terms || []);

    if (songs.length === 0) {
        return console.error("Didn't match anything");
    }

    if (args.new || args['delete-old-first']) {
        songs.sort(sortByNew);
    }

    if (args.limit && songs.length > args.limit) {
        songs.length = args.limit;
    }

    // !args['dof'] to make sure we don't uselessly sort again
    if ((args.new || args['play-new-first']) && !args['delete-old-first']) {
        songs.sort(sortByNew);
    }

    if (args['dry-paths']) {
        return console.log(
            songs.map((s) => path.join(songsPath, s)).join('\n')
        );
    }

    const playingMessage = `Playing: [${songs.length}]`;

    console.log(
        `${playingMessage}\n` +
            songs.map((e) => chalk.redBright('- ' + e)).join('\n')
    );

    exec(
        `vlc ${songs
            .map(
                (s) =>
                    `"${songsPath}/${s}" ${
                        args.new || args['play-new-first'] ? '--no-random' : ''
                    }`
            )
            .join(' ')}`
    ).catch(logErrors);
    setTimeout(() => process.exit(0), timeoutTillExit);
}

yargs(process.argv.slice(2))
    .command({
        command: '$0 [terms..]',
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
        command: ['install <id> <folder>', 'i', 'download', 'd'],
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
                });
        },
        // @ts-expect-error
        handler: ({
            folder,
            id,
            format,
            'ytdl-args': ytdlArgs,
        }: {
            folder: string;
            id: string;
            format: string;
            'ytdl-args': string;
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
                    '%(title)s.%(ext)s'
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
    .strict().argv;
