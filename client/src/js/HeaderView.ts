import { DOM } from 'client-js-lib';

import Messages from './Messages';
import ApplicationStatesModel from './ApplicationStatesModel';
import HeaderModel from './HeaderModel';
import ModalModel from './ModalModel';

export default class HeaderView {

    private readonly applicationStatesModel: ApplicationStatesModel;
    private readonly headerModel: HeaderModel;
    private readonly modalModel: ModalModel;

    private readonly $terminate: HTMLButtonElement;

    constructor(applicationStatesModel: ApplicationStatesModel, headerModel: HeaderModel, modalModel: ModalModel) {

        this.applicationStatesModel = applicationStatesModel;
        this.headerModel = headerModel;
        this.modalModel = modalModel;

        this.$terminate = DOM.query('#terminate')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
    }

    setUpEvent(): void {
        DOM.click(this.$terminate, async event => {
            event.preventDefault();          
                        
            const confirmed = await this.headerModel.terminate();
            if (confirmed) {
                this.applicationStatesModel.setTerminated();
                this.modalModel.setMessage(Messages.msg.HeaderView_001);   
            }

        });
    }
}