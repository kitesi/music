import { spawn } from 'child_process';
import { readFile, writeFile } from 'fs/promises';
import { tmpdir, EOL } from 'os';
import path from 'path';

const editor = process.env.EDITOR || 'vi';
const tmpFilePath = path.join(tmpdir(), 'music-play-list.txt');

export default async function (songs: string[]): Promise<string[]> {
    const child = spawn(editor, [tmpFilePath], {
        stdio: 'inherit',
    });

    await writeFile(tmpFilePath, songs.join(EOL));

    return new Promise((resolve, reject) => {
        child.on('exit', async (e, code) => {
            try {
                const content = await readFile(tmpFilePath, 'utf-8');
                resolve(content.split(EOL).filter((e) => e));
            } catch (err) {
                resolve([]);
            }
        });

        child.on('error', (err) => {
            reject(err);
        });
    });
}
