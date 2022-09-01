import { existsSync, readFileSync, writeFileSync } from 'fs';
import path from 'path';

interface Tag {
    name: string;
    songs: string[];
}

const tagsFileName = 'tags.json';

export function getTags(songsPath: string) {
    const filePath = path.join(songsPath, tagsFileName);

    if (!existsSync(filePath)) {
        return [];
    }

    return JSON.parse(readFileSync(filePath, 'utf-8') || '[]') as Tag[];
}

export function changeSongsInTag(
    songsPath: string,
    tagName: string,
    songs: string[],
    append: boolean
) {
    const tags = getTags(songsPath);
    const filePath = path.join(songsPath, tagsFileName);

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
