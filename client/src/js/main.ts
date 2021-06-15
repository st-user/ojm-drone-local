import ViewStateModel from './ViewStateModel';
import MainControlModel from './MainControlModel';
import MainControlView from './MainControlView';

export default function main(): void {
    window.addEventListener('DOMContentLoaded', () => {

        const viewStateModel = new ViewStateModel();
        const mainControlModel = new MainControlModel(viewStateModel);

        const mainControlView = new MainControlView(viewStateModel, mainControlModel);

        mainControlView.setUpEvent();
    });
}