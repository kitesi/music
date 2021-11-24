import Conf from 'conf';
import os from 'os';
import path from 'path';
import envPaths from 'env-paths';

import type ConfType from 'conf';

let config: ConfType<{ path: unknown; sortType: unknown }>;

try {
    config = new Conf({
        projectName: 'music-cli',
        schema: {
            path: {
                type: 'string',
                default: path.join(os.homedir(), 'Music'),
            },
            sortType: {
                type: 'string',
                // pattern: '(a|c|m)timeMs',
                enum: ['atimeMs', 'ctimeMs', 'mtimeMs'],
                default: 'mtimeMs',
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
