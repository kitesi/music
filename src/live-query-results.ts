import yargs from 'yargs';
import * as playMusic from './play-music.js';
import { getSongs } from './get-songs.js';

import type { PlayMusicArgs } from './play-music.js';

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

export async function liveQueryResults(
    { 'songs-path': songsPath, 'vlc-path': vlcPath }: PlayMusicArgs,
    timeoutTillExit: number,
    exec: (q: string) => Promise<any>
) {
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

        const songs = await getSongs(argsFromQuery, songsPath);
        const msg = songs.slice(0, 20).join('\n');

        writeToScreen(query, msg);

        lastSongs = songs;
        lastArgsFromQuery = argsFromQuery;
    });

    return new Promise<void>((res) => {
        stdin.on('end', () => res());
    });
}
