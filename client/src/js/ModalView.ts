import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import ModalModel from './ModalModel';

export default class ModalView {

    private readonly modalModel: ModalModel;

    private readonly $modal: HTMLDivElement;
    private readonly $modalDescription: HTMLDivElement;

    constructor(modalModel: ModalModel) {
        this.modalModel = modalModel;

        this.$modal = DOM.query('#modal')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$modalDescription = DOM.query('#modalDescription')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
    }

    setUpEvent(): void {
        
        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__TOGGLE_MODAL_MESSAGE, () => {
            this.render();
        });
        this.render();
    }

    private render(): void {
        const message = this.modalModel.getMessage();
        if (message) {
            this.$modal.style.display = 'flex';
            this.$modalDescription.textContent = message;
        } else {
            this.$modal.style.display = 'none';
        }
    }
}