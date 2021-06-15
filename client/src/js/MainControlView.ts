import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

import MainControlModel from './MainControlModel';
import ViewStateModel from './ViewStateModel';

export default class MainControlView {

    private readonly viewStateModel: ViewStateModel;
    private readonly mainControlModel: MainControlModel;

    private readonly $startKey: HTMLInputElement;
    private readonly $generateKey: HTMLDivElement;
    private readonly $start: HTMLDivElement;

    private readonly $takeoff: HTMLDivElement;
    private readonly $land: HTMLDivElement;

    constructor(viewStateModel: ViewStateModel, mainControlModel: MainControlModel) {

        this.viewStateModel = viewStateModel;
        this.mainControlModel = mainControlModel;

        this.$startKey = DOM.query('#startKey');
        this.$generateKey = DOM.query('#generateKey');
        this.$start = DOM.query('#start');

        this.$takeoff = DOM.query('#takeoff');
        this.$land = DOM.query('#land');
    }
    
    setUpEvent(): void {

        DOM.click(this.$generateKey, async (event: Event) => {
            event.preventDefault();

            if (!this.viewStateModel.isInit()) {
                return;
            }
    
            await this.mainControlModel.generateKey(startKey => {
                this.$startKey.value = startKey;
            });
        });

        DOM.click(this.$start, (event: Event) => {
            event.preventDefault();

            if (!this.viewStateModel.isInit()) {
                return;
            }

            this.mainControlModel.startApp(this.$startKey.value);
        });

        DOM.click(this.$takeoff, async () => {
            if (!this.viewStateModel.isLand() && !this.viewStateModel.isTakeOff()) {
                return;
            }

            await this.mainControlModel.takeoff();
        });
    
        DOM.click(this.$land, async () => {
            if (!this.viewStateModel.isLand() && !this.viewStateModel.isTakeOff()) {
                return;
            }

            await this.mainControlModel.land();
        });

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__VIEW_STATE_CHANGED, () => {
            this.render();
        });

        this.render();
    }

    private render(): void {


        if (this.viewStateModel.isInit()) {
            this.$startKey.disabled = false;
            this.enableStartButtons();
            this.disableControlButtons();
        }

        if (this.viewStateModel.isReady()) {
            this.$startKey.disabled = true;
            this.disableStartButtons();
            this.disableControlButtons();
        }

        if (this.viewStateModel.isLand()) {
            this.$startKey.disabled = true;
            this.disableStartButtons();
            this.enableControlButtons();
        }

        if (this.viewStateModel.isTakeOff()) {
            this.$startKey.disabled = true;
            this.disableStartButtons();
            this.enableControlButtons();
        }
    }




    private disableStartButtons(): void {
        this.disableElem(this.$start);
        this.disableElem(this.$generateKey);
    }

    private enableStartButtons(): void {
        this.enableElem(this.$start);
        this.enableElem(this.$generateKey);
    }

    private disableControlButtons(): void {
        this.disableElem(this.$takeoff);
        this.disableElem(this.$land);
    }

    private enableControlButtons(): void {
        this.enableElem(this.$takeoff);
        this.enableElem(this.$land);
    }

    private disableElem($elem: HTMLElement): void {
        this.resetClass($elem, 'disabled', 'enabled');
    }

    private enableElem($elem: HTMLElement): void {
        this.resetClass($elem, 'enabled', 'disabled');
    }

    private resetClass($elem: HTMLElement, classToAdd: string, classToRemove: string): void {
        $elem.classList.remove(classToRemove);
        $elem.classList.add(classToAdd);        
    }
}