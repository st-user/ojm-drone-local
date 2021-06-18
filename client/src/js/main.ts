import ViewStateModel from './ViewStateModel';
import TabView from './TabView';
import MainControlModel from './MainControlModel';
import MainControlView from './MainControlView';
import TabModel from './TabModel';
import SetupModel from './SetupModel';
import SetupView from './SetupView';
import ApplicationStatesModel from './ApplicationStatesModel';

export default function main(): void {
    window.addEventListener('DOMContentLoaded', async () => {

        const viewStateModel = new ViewStateModel();
        const tabModel = new TabModel();
        const setupModel = new SetupModel();

        const mainControlModel = new MainControlModel(viewStateModel);
        const applicationStatesModel = new ApplicationStatesModel(
            viewStateModel, tabModel, setupModel, mainControlModel
        );

        const tabView = new TabView(tabModel);
        const setupView = new SetupView(tabModel, setupModel);
        const mainControlView = new MainControlView(
            viewStateModel, applicationStatesModel, tabModel, mainControlModel
        );

        tabView.setUpEvent();
        setupView.setUpEvent();
        mainControlView.setUpEvent();

        await applicationStatesModel.init();
    });
}