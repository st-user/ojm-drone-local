import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

import Messages from './Messages';

let _sessionKey: string;
const SESSION_KEY_HTTP_HEADER_KEY = 'x-ojm-drone-local-session-key';
const SESSION_KEY_HTTP_HEADER_VALUE = {
    get: function(): string {
        return _sessionKey;
    }
};

CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__SESSION_KEY_SUCCESSFULLY_RETRIVED, event => {
    const { sessionKey } = event.detail;
    _sessionKey = sessionKey;

    CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__SESSION_KEY_AUTHORIZED_ACCESS_ENABLED);
});

function checkResponse(errorMsg?: string): (res: Response) => Response {
    return function(res: Response) {
        if (!res.ok) {
            const _msg = errorMsg || Messages.err.Common_001;
            alert(_msg);
            throw new Error('Status code is not 200');
        }
        return res;
    };
}

async function getCgi(path: string, errorMsg?: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE.get()
        }
    }).then(checkResponse(errorMsg));
}

async function deleteCgi(path: string, errorMsg?: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        method: 'DELETE',
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE.get()
        }
    }).then(checkResponse(errorMsg));
}

async function postJsonCgi(path: string, body?: BodyInit, errorMsg?: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        method: 'POST',
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE.get(),
            'Content-Type': 'application/json'
        },
        body: body
    }).then(checkResponse(errorMsg));
}

export {
    SESSION_KEY_HTTP_HEADER_KEY,
    SESSION_KEY_HTTP_HEADER_VALUE,
    getCgi,
    postJsonCgi,
    deleteCgi
};