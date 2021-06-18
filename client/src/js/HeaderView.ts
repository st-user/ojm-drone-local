import { DOM } from 'client-js-lib';
import HeaderModel from './HeaderModel';

export default class HeaderView {

    private readonly headerModel: HeaderModel;

    private readonly $terminate: HTMLButtonElement;

    constructor(headerModel: HeaderModel) {

        this.headerModel = headerModel;

        this.$terminate = DOM.query('#terminate')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
    }

    setUpEvent(): void {
        DOM.click(this.$terminate, async event => {
            event.preventDefault();          
            await this.headerModel.terminate();
        });
    }
}