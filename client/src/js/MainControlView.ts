import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

import MainControlModel from './MainControlModel';
import ViewStateModel from './ViewStateModel';
import TabModel from './TabModel';

export default class MainControlView {

    private readonly viewStateModel: ViewStateModel;
    private readonly tabModel: TabModel;
    private readonly mainControlModel: MainControlModel;

    private readonly $runArea: HTMLDivElement;

    private readonly $droneConnection: HTMLSpanElement;
    private readonly $droneBatteryLevel: HTMLSpanElement;

    private readonly $startKey: HTMLInputElement;
    private readonly $generateKey: HTMLButtonElement;
    private readonly $start: HTMLButtonElement;

    private readonly $takeoff: HTMLButtonElement;
    private readonly $land: HTMLButtonElement;

    constructor(viewStateModel: ViewStateModel, tabModel: TabModel, mainControlModel: MainControlModel) {

        this.tabModel = tabModel;
        this.viewStateModel = viewStateModel;
        this.mainControlModel = mainControlModel;

        this.$runArea = DOM.query('#runArea')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion

        this.$droneConnection = DOM.query('#droneConnection')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$droneBatteryLevel = DOM.query('#droneBatteryLevel')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion

        this.$startKey = DOM.query('#startKey')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$generateKey = DOM.query('#generateKey')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$start = DOM.query('#start')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion

        this.$takeoff = DOM.query('#takeoff')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$land = DOM.query('#land')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
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


        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__TAB_CLICKED, () => {
            this.display();
        });

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__DRONE_HEALTH_CHECKED, () => {
            this.droneHealth();
        });

        this.render();
        this.display();
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

    private display() {
        DOM.display(this.$runArea, this.tabModel.isRunSelected());
    }


    private disableStartButtons(): void {
        this.$start.disabled = true;
        this.$generateKey.disabled = true;
    }

    private enableStartButtons(): void {
        this.$start.disabled = false;
        this.$generateKey.disabled = false;
    }

    private disableControlButtons(): void {
        this.$takeoff.disabled = true;
        this.$land.disabled = true;
    }

    private enableControlButtons(): void {
        this.$takeoff.disabled = false;
        this.$land.disabled = false;
    }

    private droneHealth(): void {
        const droneHealth = this.mainControlModel.getDroneHealth();
        this.$droneConnection.textContent = droneHealth.health;
        this.$droneBatteryLevel.textContent = droneHealth.batteryLevel;       
    }
}