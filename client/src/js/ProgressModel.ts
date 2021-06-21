import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

export default class ProgressModel {

    private processing: boolean;

    constructor() {
        this.processing = false;
    }

    startProcessing(): void {
        this.setProcessing(true);
    }

    endProcessing(): void {
        this.setProcessing(false);
    }

    isProcessing(): boolean {
        return this.processing;
    }

    private setProcessing(processing: boolean): void {
        this.processing = processing;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__PROGRESS_STATE_CHANGED);
    }
}