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
import ModalModel from './ModalModel';
import ModalView from './ModalView';

export default function main(): void {
    window.addEventListener('DOMContentLoaded', async () => {

        const headerModel = new HeaderModel();
        const viewStateModel = new ViewStateModel();
        const tabModel = new TabModel();
        const setupModel = new SetupModel();
        const modalModel = new ModalModel();

        const mainControlModel = new MainControlModel(viewStateModel);
        const applicationStatesModel = new ApplicationStatesModel(
            viewStateModel, tabModel, setupModel, mainControlModel, modalModel
        );

        const headerView = new HeaderView(applicationStatesModel, headerModel, modalModel);
        const tabView = new TabView(tabModel);
        const setupView = new SetupView(tabModel, setupModel);
        const mainControlView = new MainControlView(
            viewStateModel, applicationStatesModel, tabModel, mainControlModel
        );
        const modalView = new ModalView(modalModel);

        headerView.setUpEvent();
        tabView.setUpEvent();
        setupView.setUpEvent();
        mainControlView.setUpEvent();
        modalView.setUpEvent();

        await applicationStatesModel.init();
    });
}