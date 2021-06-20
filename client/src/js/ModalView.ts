import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import ModalModel from './ModalModel';

export default class ModalView {

    private readonly modalModel: ModalModel;

    private readonly $modal: HTMLDivElement;
    private readonly $modalDescription: HTMLParagraphElement;

    private readonly $welcomeModal: HTMLDivElement;
    private readonly $welcomeModalMessage: HTMLParagraphElement;
    private readonly $welcomeModalContinue: HTMLButtonElement;
    

    constructor(modalModel: ModalModel) {
        this.modalModel = modalModel;

        this.$modal = DOM.query('#modal')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$modalDescription = DOM.query('#modalDescription')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion

        this.$welcomeModal = DOM.query('#welcomeModal')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$welcomeModalMessage = DOM.query('#welcomeModalMessage')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$welcomeModalContinue = DOM.query('#welcomeModalContinue')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        
    }

    setUpEvent(): void {

        DOM.click(this.$welcomeModalContinue, async event => {
            event.preventDefault();

            await this.modalModel.startUsingApplication();
        });
        
        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__TOGGLE_MODAL_MESSAGE, () => {
            this.render();
        });
        this.render();
    }

    private render(): void {
        this.toggleModal(
            this.$modal, this.$modalDescription, this.modalModel.getMessage()
        );
        this.toggleModal(
            this.$welcomeModal, this.$welcomeModalMessage, this.modalModel.getWelcomeModalMessage()
        );
    }

    private toggleModal($modalElem: HTMLDivElement, $modalMessageElem: HTMLParagraphElement, message: string) {
        if (message) {
            $modalElem.style.display = 'flex';
            $modalMessageElem.textContent = message;
        } else {
            $modalElem.style.display = 'none';
        }
    }
}