import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import Messages from './Messages';

export default class ModalModel {

    private readonly accessKey: string;
    private message: string;
    private welcomModalMessage: string;

    constructor() {
        this.accessKey = (DOM.query('#accessKey') as HTMLInputElement).value;
        this.message = '';
        this.welcomModalMessage = Messages.msg.ModalModel_001;
    }

    async startUsingApplication(): Promise<void> {
        await fetch('/dmz/startUsingApplication', {
            headers: {
                'x-ojm-drone-local-access-key': `${this.accessKey}`
            }
        }).then(res => {
            if (res.ok) {
                return res.json();
            }
            alert(Messages.err.ModalModel_001);
            throw new Error('The status code is not 200.');
        }).then(json => {
            const sessionKey: string = json.sessionKey;
            
            this.setWelcomeModalMessage('');
            CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__SESSION_KEY_SUCCESSFULLY_RETRIVED, {
                sessionKey
            });
            CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__TOGGLE_MODAL_MESSAGE);

        }).catch(console.error);
    }

    setMessage(message: string): void {
        this.message = message;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__TOGGLE_MODAL_MESSAGE);
    }

    getMessage(): string {
        return this.message;
    }

    setWelcomeModalMessage(message: string): void {
        this.welcomModalMessage = message;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__TOGGLE_MODAL_MESSAGE);
    }

    getWelcomeModalMessage(): string {
        return this.welcomModalMessage;
    }
}