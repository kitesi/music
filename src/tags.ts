import { existsSync, readFileSync, writeFileSync } from 'fs';
import path from 'path';

import { config } from './config.js';

const songsPath = config.get('path') as string;
const filePath = path.join(songsPath, 'tags.json');

interface Tag {
    name: string;
    songs: string[];
}

export function getTags() {
    if (!existsSync(filePath)) {
        return [];
    }

    return JSON.parse(readFileSync(filePath, 'utf-8') || '[]') as Tag[];
}

export function changeSongsInTag(
    tagName: string,
    songs: string[],
    append: boolean
) {
    const tags = getTags();
    let tag = tags.find((t) => t.name === tagName);

    if (!tag) {
        tags.push({
            name: tagName,
            songs,
        });
    } else if (!append) {
        tag.songs = songs;
    } else {
        for (const song of songs) {
            if (!tag.songs.includes(song)) {
                tag.songs.push(song);
            }
        }
    }

    writeFileSync(filePath, JSON.stringify(tags), {});
}
