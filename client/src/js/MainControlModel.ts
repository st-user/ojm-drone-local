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
        await getCgi('/generateKey')
            .then(res => res.json())
            .then(ret => {
                this.setStartKeyWithEvent(ret.startKey);
            })
            .catch(e => {
                console.error(e);
                alert('Can not generate key. Remote server may fail to authorize me or be unavailable.');
            });
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

        await postJsonCgi('/startApp', JSON.stringify({ startKey }))
            .then(res => {
                if (res.ok) {
                    return res.json();
                }
                throw new Error('Request does not success.');
            })
            .then(() => {
                this.viewStateModel.toReady();

            })
            .catch(e => {
                console.error(e);
                alert('Can not start signaling. Remote server may fail to validate the code or be unavailable.');
                this.viewStateModel.toInit();
            });
    }

    async stopApp(): Promise<void> {
        let msg = 'Are you sure you want to stop the application?';
        msg += ' If you terminate the application, the video streaming stops and drone lands (if it has already taken off).';

        if (confirm(msg)) {
            await postJsonCgi('/stopApp').then(res => {
                if (res.ok) {
                    this.viewStateModel.toInit();
                    return;
                }
                throw new Error('Request does not success.');
            });
        }

    }

    async takeoff(): Promise<void> {
        this.viewStateModel.toTakeOff();
        await postJsonCgi('/takeoff').then(res => res.json());
    }

    async land(): Promise<void> {
        this.viewStateModel.toLand();
        await postJsonCgi('/land').then(res => res.json());
    }
}