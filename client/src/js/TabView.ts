import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import TabModel from './TabModel';

export default class TabView {

    private readonly tabModel: TabModel;

    private readonly $setup: HTMLLinkElement;
    private readonly $run: HTMLLinkElement;

    constructor(tabModel: TabModel) {
        this.tabModel = tabModel;
        this.$setup = DOM.query('#setupTab a')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$run = DOM.query('#runTab a')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
    }


    setUpEvent(): void {

        DOM.click(this.$setup, event => {
            event.preventDefault();
            this.tabModel.setup();
        });

        DOM.click(this.$run, event => {
            event.preventDefault();
            this.tabModel.run();
        });

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__TAB_CLICKED, () => {
            this.render();
        });

        this.render();
    }

    private render(): void {
        this.$setup.setAttribute('aria-selected', String(this.tabModel.isSetupSelected()));
        this.$run.setAttribute('aria-selected', String(this.tabModel.isRunSelected()));
    }
}