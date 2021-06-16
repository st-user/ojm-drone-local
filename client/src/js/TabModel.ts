import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

enum Tab {
    Setup,
    Run
}

export default class TabModel {

    private selectedTab: Tab

    constructor() {
        this.selectedTab = Tab.Setup;
    }

    isSetupSelected(): boolean {
        return this.selectedTab == Tab.Setup;
    }

    isRunSelected(): boolean {
        return this.selectedTab == Tab.Run;
    }

    setup(): void {
        this.select(Tab.Setup);
    }

    run(): void {
        this.select(Tab.Run);
    }

    private select(tab: Tab) {
        this.selectedTab = tab;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__TAB_CLICKED);
    }

}