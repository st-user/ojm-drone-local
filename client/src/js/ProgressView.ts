import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import ProgressModel from './ProgressModel';

export default class ProgressView {

    private readonly progressModel: ProgressModel;

    private readonly $progressBar: HTMLDivElement;

    constructor(progressModel: ProgressModel) {
        this.progressModel = progressModel;
        
        this.$progressBar = DOM.query('#progressBar')!;  // eslint-disable-line @typescript-eslint/no-non-null-assertion
    }

    setUpEvent(): void {

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__PROGRESS_STATE_CHANGED, () => {
            this.render();
        });

        this.render();
    }

    private render(): void {
        DOM.display(this.$progressBar, this.progressModel.isProcessing());
    }
}