import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

export default class ModalModel {

    private message: string;

    constructor() {
        this.message = '';
    }

    setMessage(message: string): void {
        this.message = message;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__TOGGLE_MODAL_MESSAGE);
    }

    getMessage(): string {
        return this.message;
    }
}