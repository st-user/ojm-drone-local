import { DOM } from 'client-js-lib';

const SESSION_KEY_HTTP_HEADER_KEY = 'x-ojm-drone-local-session-key';
const SESSION_KEY_HTTP_HEADER_VALUE = (DOM.query('#sessionKey') as HTMLInputElement).value;

function checkResponse(errorMsg?: string): (res: Response) => Response {
    return function(res: Response) {
        if (!res.ok) {
            const _msg = errorMsg || 'The appllication failed to complete the process. Please check whether the application is running.';
            alert(_msg);
            throw new Error('Status code is not 200');
        }
        return res;
    };
}

async function getCgi(path: string, errorMsg?: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE
        }
    }).then(checkResponse(errorMsg));
}

async function deleteCgi(path: string, errorMsg?: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        method: 'DELETE',
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE
        }
    }).then(checkResponse(errorMsg));
}

async function postJsonCgi(path: string, body?: BodyInit, errorMsg?: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        method: 'POST',
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE,
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