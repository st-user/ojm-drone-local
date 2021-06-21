import { CommonEventDispatcher, DOM } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

import MainControlModel from './MainControlModel';
import ApplicationStatesModel from './ApplicationStatesModel';
import { DroneHealthState, BatteryLevelWarningState} from './ApplicationStatesModel';
import ViewStateModel from './ViewStateModel';
import TabModel from './TabModel';

const HEALTH_STATES_CLASSES = ['is-ok', 'is-ng', 'is-warn'];



export default class MainControlView {

    private readonly viewStateModel: ViewStateModel;
    private readonly applicationStatesModel: ApplicationStatesModel;
    private readonly tabModel: TabModel;
    private readonly mainControlModel: MainControlModel;

    private readonly $runArea: HTMLDivElement;

    private readonly $droneConnection: HTMLSpanElement;
    private readonly $droneBatteryLevel: HTMLSpanElement;

    private readonly $startKey: HTMLInputElement;
    private readonly $start: HTMLButtonElement;
    private readonly $stop: HTMLButtonElement;
    private readonly $generateKey: HTMLButtonElement;

    private readonly $takeoff: HTMLButtonElement;
    private readonly $land: HTMLButtonElement;

    constructor(viewStateModel: ViewStateModel, applicationStatesModel: ApplicationStatesModel, tabModel: TabModel, mainControlModel: MainControlModel) {

        this.viewStateModel = viewStateModel;
        this.applicationStatesModel = applicationStatesModel;
        this.tabModel = tabModel;
        this.mainControlModel = mainControlModel;

        this.$runArea = DOM.query('#runArea')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion

        this.$droneConnection = DOM.query('#droneConnection')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$droneBatteryLevel = DOM.query('#droneBatteryLevel')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion

        this.$startKey = DOM.query('#startKey')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$start = DOM.query('#start')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$stop = DOM.query('#stop')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$generateKey = DOM.query('#generateKey')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion

        this.$takeoff = DOM.query('#takeoff')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
        this.$land = DOM.query('#land')!; // eslint-disable-line @typescript-eslint/no-non-null-assertion
    }
    
    setUpEvent(): void {

        DOM.click(this.$generateKey, async (event: Event) => {
            event.preventDefault();

            if (!this.viewStateModel.isInit()) {
                return;
            }
    
            await this.mainControlModel.generateKey();
        });

        DOM.keyup(this.$startKey, event => {
            event.preventDefault();
            this.mainControlModel.setStartKeyWithEvent(this.$startKey.value);
        });

        DOM.click(this.$start, (event: Event) => {
            event.preventDefault();

            if (!this.viewStateModel.isInit()) {
                return;
            }
            this.mainControlModel.setStartKeyNoEvent(this.$startKey.value);
            this.mainControlModel.startApp();
        });

        DOM.click(this.$stop, (event: Event) => {
            event.preventDefault();
            this.mainControlModel.stopApp();
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

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__START_KEY_INPUT_STATE_CHANGED, () => {
            this.render();
        });


        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__TAB_CLICKED, () => {
            this.display();
        });

        CommonEventDispatcher.on(CustomEventNames.OJM_DRONE_LOCAL__DRONE_HEALTH_CHECKED, () => {
            this.droneHealth();
        });
        
        this.display();
    }

    private render(): void {

        this.$startKey.value = this.mainControlModel.getStartKey();

        this.$startKey.disabled = !this.mainControlModel.canInputStartKey();
        this.$start.disabled = !this.mainControlModel.canStart();
        this.$stop.disabled = !this.mainControlModel.canStop();
        this.$generateKey.disabled = !this.mainControlModel.canGenerate();
        this.$takeoff.disabled = !this.mainControlModel.canTakeOff();
        this.$land.disabled = !this.mainControlModel.canLand();

    }

    private display(): void {
        DOM.display(this.$runArea, this.tabModel.isRunSelected());
    }

    private droneHealth(): void {
        this.resetClass(this.$droneConnection, ...HEALTH_STATES_CLASSES);
        this.resetClass(this.$droneBatteryLevel, ...HEALTH_STATES_CLASSES);

        const droneHealth = this.applicationStatesModel.getDroneHealth();

        const healthInfo = droneHealth.getHealthInfo();

        if (healthInfo.state === DroneHealthState.Ng) {
            this.$droneConnection.classList.add('is-ng');
        }
        if (healthInfo.state === DroneHealthState.Ok) {
            this.$droneConnection.classList.add('is-ok');
        } 
        this.$droneConnection.textContent = healthInfo.desc;

        const batteryLevelInfo = droneHealth.getBatteryLevelInfo();
        if (batteryLevelInfo.state === BatteryLevelWarningState.Low) {
            this.$droneBatteryLevel.classList.add('is-ng');
        }
        if (batteryLevelInfo.state === BatteryLevelWarningState.Middle) {
            this.$droneBatteryLevel.classList.add('is-warn');
        }
        if (batteryLevelInfo.state === BatteryLevelWarningState.High) {
            this.$droneBatteryLevel.classList.add('is-ok');
        }
        this.$droneBatteryLevel.textContent = batteryLevelInfo.desc;       
    }

    private resetClass($elem: HTMLElement, ...classes: string[]) {
        classes.forEach(cls => {
            $elem.classList.remove(cls);
        });
    }
}