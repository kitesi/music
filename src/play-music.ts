import type { Argv } from 'yargs';

import chalk from 'chalk';

export interface PlayMusicArgs {
    terms?: string[];
    limit: number;
    new: boolean;
    persist?: boolean;
    live?: boolean;
    'dry-run': boolean;
    'dry-paths': boolean;
    'play-new-first': boolean;
    'delete-old-first': boolean;
    'vlc-path': string;
    'songs-path': string;
    'sort-type': 'a' | 'c' | 'm';
}

export function builder(y: Argv) {
    return y
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
        .option('live', {
            type: 'boolean',
            describe: 'get live query results with stdin input',
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
        });
}

interface RunArgs {
    songs: string[];
    args: PlayMusicArgs;
    exec: (query: string) => Promise<any>;
    songsPath: string;
    vlcPath: string;
}

export function run({ exec, vlcPath, args, songs, songsPath }: RunArgs) {
    exec(
        `${vlcPath} ${songs
            .map(
                (s) =>
                    `"${songsPath}/${s}" ${
                        args.new || args['play-new-first'] ? '--no-random' : ''
                    }`
            )
            .join(' ')}`
    ).catch((reason: any) =>
        console.error('Error: ' + (reason?.message || `\n\n${reason}`))
    );
}

export function message(songs: string[]) {
    const playingMessage = `Playing: [${songs.length}]`;

    console.log(
        `${playingMessage}\n` +
            songs.map((e) => chalk.redBright('- ' + e)).join('\n')
    );
}
