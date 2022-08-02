import Conf from 'conf';
import os from 'os';
import path from 'path';
import envPaths from 'env-paths';

import type ConfType from 'conf';

let config: ConfType<{
    path: unknown;
    sortType: unknown;
    pathToVLC: unknown;
    persist: unknown;
}>;

try {
    config = new Conf({
        projectName: 'music-cli',
        schema: {
            path: {
                type: 'string',
                default: path.join(os.homedir(), 'Music'),
            },
            pathToVLC: {
                type: 'string',
                // assume it's in global path
                default: 'vlc',
            },
            sortType: {
                type: 'string',
                enum: ['atimeMs', 'ctimeMs', 'mtimeMs'],
                default: 'mtimeMs',
            },
            persist: {
                type: 'boolean',
                default: false,
            },
        },
    });
} catch (err) {
    console.error(err);
    console.log(
        'Config file located at ' +
            path.join(envPaths('music-cli').config, 'config.json')
    );
    process.exit(1);
}

export { config };
