import { DOM } from 'client-js-lib';

const SESSION_KEY_HTTP_HEADER_KEY = 'x-ojm-drone-local-session-key';
const SESSION_KEY_HTTP_HEADER_VALUE = (DOM.query('#sessionKey') as HTMLInputElement).value;

async function getCgi(path: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE
        }
    });
}

async function deleteCgi(path: string): Promise<Response> {
    return await fetch('/cgi' + path, {
        method: 'DELETE',
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE
        }
    });
}

async function postJsonCgi(path: string, body?: BodyInit): Promise<Response> {
    return await fetch('/cgi' + path, {
        method: 'POST',
        headers: {
            [SESSION_KEY_HTTP_HEADER_KEY]: SESSION_KEY_HTTP_HEADER_VALUE,
            'Content-Type': 'application/json'
        },
        body: body
    });
}

export {
    SESSION_KEY_HTTP_HEADER_KEY,
    SESSION_KEY_HTTP_HEADER_VALUE,
    getCgi,
    postJsonCgi,
    deleteCgi
};