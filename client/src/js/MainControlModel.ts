import { CommonEventDispatcher } from 'client-js-lib';
import { getCgi, postJsonCgi } from './Auth';
import { CustomEventNames } from './CustomEventNames';
import ViewStateModel from './ViewStateModel';

export default class MainControlModel {

    private readonly viewStateModel: ViewStateModel;

    private startKey: string

    constructor(viewStateModel: ViewStateModel) {
        this.viewStateModel = viewStateModel;
        this.startKey = '';
    }

    async generateKey(): Promise<void> {
        await getCgi('/generateKey', 'Can not generate key. Remote server may fail to authorize me or be unavailable.')
            .then(res => res.json())
            .then(ret => {
                this.setStartKeyWithEvent(ret.startKey);
            }).catch(console.error);
    }

    setStartKeyNoEvent(startKey: string): void {
        this.startKey = startKey;
    }

    setStartKeyWithEvent(startKey: string): void {
        this.startKey = startKey;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__START_KEY_INPUT_STATE_CHANGED);
    }

    getStartKey(): string {
        return this.startKey;
    }

    async startApp(): Promise<void> {
        const startKey = this.startKey;

        await postJsonCgi('/startApp', JSON.stringify({ startKey }), 'Can not start signaling. Remote server may fail to validate the code or be unavailable.')
            .then(res => res.json())
            .then(() => {
                this.viewStateModel.toReady();
            })
            .catch(e => {
                console.error(e);
                this.viewStateModel.toInit();
            });
    }

    async stopApp(): Promise<void> {
        let msg = 'Are you sure you want to stop the application?';
        msg += ' If you terminate the application, the video streaming stops and drone lands (if it has already taken off).';

        if (confirm(msg)) {
            await postJsonCgi('/stopApp').then(() => {
                this.viewStateModel.toInit();
            });
        }

    }

    async takeoff(): Promise<void> {
        this.viewStateModel.toTakeOff();
        await postJsonCgi('/takeoff');
    }

    async land(): Promise<void> {
        this.viewStateModel.toLand();
        await postJsonCgi('/land');
    }
}