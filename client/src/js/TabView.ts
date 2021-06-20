import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import TabModel from './TabModel';

export default class TabView {

    private readonly tabModel: TabModel;

    private readonly $setup: HTMLLIElement;
    private readonly $run: HTMLLIElement;

    constructor(tabModel: TabModel) {
        this.tabModel = tabModel;
        this.$setup = DOM.query('#setupTab')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$run = DOM.query('#runTab')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
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
        this.toggleActive(this.$setup, this.tabModel.isSetupSelected());
        this.toggleActive(this.$run, this.tabModel.isRunSelected());
    }

    private toggleActive($tab: HTMLLIElement, isActive: boolean) {
        $tab.classList.remove('is-active');
        if (isActive) {
            $tab.classList.add('is-active');
        }        
    }
}