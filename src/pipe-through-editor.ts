import { spawn } from 'child_process';
import { readFile, writeFile } from 'fs/promises';
import { tmpdir, EOL } from 'os';
import path from 'path';

import openGraphScraper from 'open-graph-scraper';

const editor = process.env.EDITOR || 'vi';

async function baseEditTmpFile(
    tmpFilePath: string,
    content: string
): Promise<string> {
    const child = spawn(editor, [tmpFilePath], {
        stdio: 'inherit',
    });

    await writeFile(tmpFilePath, content);

    return new Promise((resolve, reject) => {
        child.on('exit', async (e, code) => {
            try {
                const content = await readFile(tmpFilePath, 'utf-8');
                resolve(content);
            } catch (err) {
                resolve('');
            }
        });

        child.on('error', (err) => {
            reject(err);
        });
    });
}

export async function editSongList(songs: string[]): Promise<string[]> {
    const tmpFilePath = path.join(tmpdir(), 'music-play-list.txt');
    return baseEditTmpFile(tmpFilePath, songs.join(EOL)).then((content) =>
        content.split(EOL).filter((e) => e)
    );
}

export async function editSongInstallName(link: string): Promise<string> {
    const tmpFilePath = path.join(tmpdir(), 'music-install-name');
    const { result, error } = await openGraphScraper({ url: link });
    // @ts-ignore
    const title = (result.ogTitle as string) || '%(title)s.%(ext)s';

    if (error) {
        return new Promise((resolve, reject) => {
            reject(error);
        });
    }

    return baseEditTmpFile(tmpFilePath, title).then((content) =>
        content.trim()
    );
}
