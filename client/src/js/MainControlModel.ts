import { CommonEventDispatcher } from 'client-js-lib';

import { getCgi, postJsonCgi } from './AuthorizedAccess';
import Messages from './Messages';
import { CustomEventNames } from './CustomEventNames';
import ViewStateModel from './ViewStateModel';
import ProgressModel from './ProgressModel';

export default class MainControlModel {

    private readonly progressModel: ProgressModel;
    private readonly viewStateModel: ViewStateModel;

    private startKey: string

    constructor(progressModel: ProgressModel, viewStateModel: ViewStateModel) {
        this.progressModel = progressModel;
        this.viewStateModel = viewStateModel;
        this.startKey = '';
    }

    async generateKey(): Promise<void> {
        this.progressModel.startProcessing();
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__START_KEY_INPUT_STATE_CHANGED);

        await getCgi('/generateKey', Messages.err.MainControlModel_001)
            .then(res => res.json())
            .then(ret => {

                this.progressModel.endProcessing();
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

        this.progressModel.startProcessing();
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__START_KEY_INPUT_STATE_CHANGED);

        await postJsonCgi('/startApp', JSON.stringify({ startKey }), Messages.err.MainControlModel_002)
            .then(res => res.json())
            .then(() => {
                this.progressModel.startProcessing();
                this.viewStateModel.toReady();
            })
            .catch(e => {
                console.error(e);
                this.progressModel.endProcessing();
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

    canInputStartKey(): boolean {
        if (this.progressModel.isProcessing()) {
            return false;
        }
        return this.viewStateModel.isInit();
    }

    canStart(): boolean {
        if (this.progressModel.isProcessing()) {
            return false;
        }
        return this.viewStateModel.isInit() && !!this.startKey;
    }

    canStop(): boolean {
        return !this.viewStateModel.isInit();
    }

    canGenerate(): boolean {
        if (this.progressModel.isProcessing()) {
            return false;
        }
        return this.viewStateModel.isInit();
    }

    canTakeOff(): boolean {
        if (this.progressModel.isProcessing()) {
            return false;
        }
        return this.viewStateModel.isTakeOff() || this.viewStateModel.isLand();
    }

    canLand(): boolean {
        if (this.progressModel.isProcessing()) {
            return false;
        }
        return this.viewStateModel.isTakeOff() || this.viewStateModel.isLand();
    }
}