import ViewStateModel from './ViewStateModel';
import TabView from './TabView';
import MainControlModel from './MainControlModel';
import MainControlView from './MainControlView';
import TabModel from './TabModel';
import SetupModel from './SetupModel';
import SetupView from './SetupView';

export default function main(): void {
    window.addEventListener('DOMContentLoaded', async () => {

        const viewStateModel = new ViewStateModel();
        const tabModel = new TabModel();

        const setupModel = new SetupModel();
        const mainControlModel = new MainControlModel(viewStateModel);

        const tabView = new TabView(tabModel);
        const setupView = new SetupView(tabModel, setupModel);
        const mainControlView = new MainControlView(viewStateModel, tabModel, mainControlModel);

        tabView.setUpEvent();
        setupView.setUpEvent();
        mainControlView.setUpEvent();

        await setupModel.init();
        if (setupModel.getSavedAccessTokenDesc()) {
            tabModel.run();
        }
        await mainControlModel.init();
    });
}