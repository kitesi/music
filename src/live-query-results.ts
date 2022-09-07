import yargs from 'yargs';
import * as playMusic from './play-music.js';
import { getSongs } from './get-songs.js';

import type { PlayMusicArgs } from './play-music.js';

function writeToScreen(query: string, msg: string, songs?: string[]) {
    process.stdout.write('\r');
    process.stdout.clearScreenDown();

    const queryMessage = 'Search: ' + query;

    process.stdout.write(
        queryMessage + '\n-----------------------\n' + msg + '\n'
    );

    let lines = msg.split('\n').length;

    if (songs) {
        lines += songs.filter((s) => s.length > process.stdout.columns).length;
    }

    process.stdout.moveCursor(queryMessage.length, -(lines + 2));
}

export async function liveQueryResults(
    { songsPath, vlcPath }: PlayMusicArgs,
    timeoutTillExit: number,
    exec: (q: string) => Promise<any>
) {
    const { stdin } = process;
    const wordsRegex = /[\s+#,]/g;

    stdin.setRawMode(true);
    stdin.resume();
    stdin.setEncoding('utf8');

    const parser = yargs()
        // idk the typing is off for yargs
        .command({
            command: '$0 [terms..]',
            describe: '',
            builder: playMusic.builder,
            handler: () => {},
        })
        .help(false)
        .strict(true);

    let query = '';
    let lastSongs: string[] = [];
    let lastArgsFromQuery: ReturnType<ReturnType<typeof parser>['parseSync']>;

    writeToScreen('', '');

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

                playMusic.execVLC({
                    // @ts-ignore
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
            .parseSync(query);

        if (hasError) {
            return;
        }

        argsFromQuery['songs-path'] = songsPath;

        // @ts-ignore
        const songs = await getSongs(argsFromQuery, songsPath);
        const showenSongs = songs.slice(0, 20);
        const msg = showenSongs.join('\n');

        writeToScreen(query, msg, showenSongs);

        lastSongs = songs;
        lastArgsFromQuery = argsFromQuery;
    });

    return new Promise<void>((res) => {
        stdin.on('end', () => res());
    });
}
