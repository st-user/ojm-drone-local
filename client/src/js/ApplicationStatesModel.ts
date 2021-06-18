import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

import TabModel from './TabModel';
import MainControlModel from './MainControlModel';
import SetupModel from './SetupModel';

import ViewStateModel from './ViewStateModel';

const DRONE_HEALTH_DESCS = ['-', 'OK', 'NG'];

const STATE_CONNECTION_RETRY_INTERVAL_MILLIS = 1000;
const STATE_CONNECTION_MAX_RETRY = 10;

enum BatteryLevelWarningState {
    Unknown,
    Low,
    Middle,
    High
}

enum DroneHealthState {
    Unknown,
	Ok,
	Ng,
}

enum DroneState {
    Unknown,
    Ready,
	Land,
	TakeOff,
}

class DroneHealth {

    private _health: DroneHealthState;
    private _batteryLevel: BatteryLevelWarningState;

    constructor() {
        this._health = DroneHealthState.Unknown;
        this._batteryLevel = BatteryLevelWarningState.Unknown;
    }

    setData(_health: number, _batteryLevel: number): void {
        this._health = _health;
        this._batteryLevel = _batteryLevel;
    }

    getHealthInfo(): { state: DroneHealthState, desc: string } {
        return { state: this._health, desc:  DRONE_HEALTH_DESCS[this._health] || '-' };
    }

    getBatteryLevelInfo(): { state: BatteryLevelWarningState, desc: string } {

        if (this._health !== DroneHealthState.Ok) {
            return { state: BatteryLevelWarningState.Unknown, desc: '-%' };

        }

        if (this._batteryLevel <= 20) {
            return { state: BatteryLevelWarningState.Low, desc: `${this._batteryLevel}%` };
        }

        if (this._batteryLevel <= 50) {
            return { state: BatteryLevelWarningState.Middle, desc: `${this._batteryLevel}%` };
        }

        return { state: BatteryLevelWarningState.High, desc: `${this._batteryLevel}%` };
    }
}

export { DroneHealthState, BatteryLevelWarningState };

enum ApplicationState {
    Init,
    Started
}
type StatesResp = { accessTokenDesc: string, applicationState: number, startKey: string };

export default class ApplicationStatesModel {
     
    private readonly viewStateModel: ViewStateModel;
    private readonly tabModel: TabModel;
    private readonly setupModel: SetupModel
    private readonly mainControlModel: MainControlModel;

    private applicationState: ApplicationState;
    private readonly droneHealth: DroneHealth;



    private retryTimer: any; // eslint-disable-line @typescript-eslint/no-explicit-any
    private retryCount: number;

    constructor(viewStateModel: ViewStateModel, tabModel: TabModel, setupModel: SetupModel, mainControlModel: MainControlModel) {
        this.viewStateModel = viewStateModel;
        this.tabModel = tabModel;
        this.setupModel = setupModel;
        this.mainControlModel = mainControlModel;

        this.applicationState = ApplicationState.Init;
        this.droneHealth = new DroneHealth();
        this.retryTimer = undefined;
        this.retryCount = 0;
    }

    async init(): Promise<void> {

        const statesResp: StatesResp = await fetch('/checkApplicationStates')
            .then(res => res.json());

        this.applicationState = statesResp.applicationState;
        this.setupModel.setSavedAccessTokenDesc(statesResp.accessTokenDesc);
        this.mainControlModel.setStartKeyWithEvent(statesResp.startKey);
        
        if (this.setupModel.getSavedAccessTokenDesc()) {
            this.tabModel.run();
        }

        this.initStatesClient(true);
    }

    private initStatesClient(startAppOnOpen: boolean): void {
        const wsProtocol = 0 <= location.protocol.indexOf('https') ? 'wss' : 'ws';
        const websocket = new WebSocket(`${wsProtocol}://${location.host}/state`);
        websocket.onmessage = (event: MessageEvent) => {

            const dataJson = JSON.parse(event.data);
            const messageType = dataJson.messageType;

            switch(messageType) {
            case 'droneInfo':

                this.droneHealth.setData(
                    dataJson.healths.health, dataJson.healths.batteryLevel
                );

                switch(dataJson.state) {
                case DroneState.Ready:
                    this.viewStateModel.toReady();
                    break;

                case DroneState.Land:
                    if (this.droneHealth.getHealthInfo().state === DroneHealthState.Ok) {
                        this.viewStateModel.toLand();
                    } else {
                        this.viewStateModel.toReady();
                    }
                    break;
                default:
                }


                CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__DRONE_HEALTH_CHECKED);

                break;
            default:
                return;
            }
        };

        websocket.onopen = () => {

            console.log('open');
            clearTimeout(this.retryTimer);
            this.retryCount = 0;

            if (startAppOnOpen && this.applicationState === ApplicationState.Started) {
                this.mainControlModel.startApp();
            }
        };

        websocket.onclose = () => {
            this.retry();
        };
    }

    private retry() {
        clearTimeout(this.retryTimer);
        this.retryTimer = setTimeout(() => {
            this.retryCount++;
            if (STATE_CONNECTION_MAX_RETRY < this.retryCount) {
                clearTimeout(this.retryTimer);
                this.retryCount = 0;
                alert('Can not start application. Remote server may be unavailable.');
                return;
            }
            this.initStatesClient(false);
            this.retry();
        }, STATE_CONNECTION_RETRY_INTERVAL_MILLIS);
    }

    getDroneHealth(): DroneHealth {
        return this.droneHealth;
    }
}