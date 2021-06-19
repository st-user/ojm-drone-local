import { CommonEventDispatcher } from 'client-js-lib';

import { getCgi, postJsonCgi } from './Auth';
import Messages from './Messages';
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
        await getCgi('/generateKey', Messages.err.MainControlModel_001)
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

        await postJsonCgi('/startApp', JSON.stringify({ startKey }), Messages.err.MainControlModel_002)
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
        if (confirm(Messages.msg.MainControlModel_001)) {
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