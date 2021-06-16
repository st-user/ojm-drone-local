import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

import TabModel from './TabModel';
import SetupModel from './SetupModel';


export default class SetupView {

    private readonly tabModel: TabModel;
    private readonly setupModel: SetupModel;

    private readonly $setupArea: HTMLDivElement;
    private readonly $accessToken: HTMLInputElement;
    private readonly $savedAccessTokenDesc: HTMLSpanElement;
    private readonly $updateAccessToken: HTMLButtonElement;
    private readonly $deleteAccessToken: HTMLButtonElement;


    constructor(tabModel: TabModel, setupModel: SetupModel) {

        this.tabModel = tabModel;
        this.setupModel = setupModel;

        this.$setupArea = DOM.query('#setupArea')!;  // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$accessToken = DOM.query('#accessToken')!;  // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$savedAccessTokenDesc = DOM.query('#savedAccessTokenDesc')!;  // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$updateAccessToken = DOM.query('#updateAccessToken')!;  // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$deleteAccessToken = DOM.query('#deleteAccessToken')!;  // eslint-disable-line @typescript-eslint/no-non-null-assertion
    }

    setUpEvent(): void {

        DOM.keyup(this.$accessToken, event => {
            event.preventDefault();
            this.setupModel.setAccessToken(this.$accessToken.value);
        });

        DOM.click(this.$updateAccessToken, async event => {
            event.preventDefault();
            this.setupModel.setAccessToken(this.$accessToken.value);
            await this.setupModel.update();
        });

        DOM.click(this.$deleteAccessToken, async event => {
            event.preventDefault();
            await this.setupModel.delete();
        });

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__TAB_CLICKED, () => {
            this.display();
        });

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED, () => {
            this.render();
        });

        this.display();
    }

    private render() {
        this.$updateAccessToken.disabled = !this.setupModel.canUpdate();
        this.$deleteAccessToken.disabled = !this.setupModel.canDelete();

        this.$accessToken.value = this.setupModel.getAccessToken();
        this.$savedAccessTokenDesc.textContent = this.setupModel.getSavedAccessTokenDesc();

    }

    private display() {
        DOM.display(this.$setupArea, this.tabModel.isSetupSelected());
    }
}