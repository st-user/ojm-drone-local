import ViewStateModel from './ViewStateModel';
import TabView from './TabView';
import MainControlModel from './MainControlModel';
import MainControlView from './MainControlView';
import TabModel from './TabModel';
import SetupModel from './SetupModel';
import SetupView from './SetupView';
import ApplicationStatesModel from './ApplicationStatesModel';
import HeaderModel from './HeaderModel';
import HeaderView from './HeaderView';

export default function main(): void {
    window.addEventListener('DOMContentLoaded', async () => {

        const headerModel = new HeaderModel();
        const viewStateModel = new ViewStateModel();
        const tabModel = new TabModel();
        const setupModel = new SetupModel();

        const mainControlModel = new MainControlModel(viewStateModel);
        const applicationStatesModel = new ApplicationStatesModel(
            viewStateModel, tabModel, setupModel, mainControlModel
        );

        const headerView = new HeaderView(headerModel);
        const tabView = new TabView(tabModel);
        const setupView = new SetupView(tabModel, setupModel);
        const mainControlView = new MainControlView(
            viewStateModel, applicationStatesModel, tabModel, mainControlModel
        );

        headerView.setUpEvent();
        tabView.setUpEvent();
        setupView.setUpEvent();
        mainControlView.setUpEvent();

        await applicationStatesModel.init();
    });
}