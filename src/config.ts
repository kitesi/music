import Conf from 'conf';
import os from 'os';
import path from 'path';

const config = new Conf({
    projectName: 'music-cli',
    schema: {
        path: {
            type: 'string',
            default: path.join(os.homedir(), 'Music'),
        },
    },
});

export { config };
