import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

import TabModel from './TabModel';
import MainControlModel from './MainControlModel';
import SetupModel from './SetupModel';

import ViewStateModel from './ViewStateModel';
import ModalModel from './ModalModel';

import { SESSION_KEY_HTTP_HEADER_VALUE, getCgi } from './Auth';

const DRONE_HEALTH_DESCS = ['-', 'OK', 'NG'];

const STATE_CONNECTION_RETRY_INTERVAL_MILLIS = 1000;
const STATE_CONNECTION_MAX_RETRY = 10;

enum ApplicationState {
    Init,
    Started,
    Terminated
}
type StatesResp = { accessTokenDesc: string, applicationState: number, startKey: string };

enum BatteryLevelWarningState {
    Unknown,
    Low,
    Middle,
    High
}

enum DroneHealthState {
    Unknown,
	Ok,
	Ng
}

enum DroneState {
    Init,
    Ready,
	Land,
	TakeOff
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

export default class ApplicationStatesModel {
     
    private readonly viewStateModel: ViewStateModel;
    private readonly tabModel: TabModel;
    private readonly setupModel: SetupModel
    private readonly mainControlModel: MainControlModel;
    private readonly modalModel: ModalModel;

    private readonly sessionKey: string;
    private applicationState: ApplicationState;
    private readonly droneHealth: DroneHealth;

    private websocket: WebSocket | undefined;

    private retryTimer: any; // eslint-disable-line @typescript-eslint/no-explicit-any
    private retryCount: number;

    constructor(viewStateModel: ViewStateModel, tabModel: TabModel, setupModel: SetupModel, mainControlModel: MainControlModel, modalModel: ModalModel) {
        this.viewStateModel = viewStateModel;
        this.tabModel = tabModel;
        this.setupModel = setupModel;
        this.mainControlModel = mainControlModel;
        this.modalModel = modalModel;

        this.sessionKey = SESSION_KEY_HTTP_HEADER_VALUE;
        this.applicationState = ApplicationState.Init;
        this.droneHealth = new DroneHealth();

        this.websocket = undefined;
        this.retryTimer = undefined;
        this.retryCount = 0;
    }

    async init(): Promise<void> {

        const statesResp: StatesResp = await getCgi('/checkApplicationStates')
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
        this.websocket = new WebSocket(`${wsProtocol}://${location.host}/cgi/state?sessionKey=${SESSION_KEY_HTTP_HEADER_VALUE}`);
        this.websocket.onmessage = (event: MessageEvent) => {

            if (this.applicationState === ApplicationState.Terminated) {
                return;
            }

            const dataJson = JSON.parse(event.data);
            const messageType = dataJson.messageType;

            switch(messageType) {
            case 'checkSessionKey':
                this.detectServerStopping(dataJson.currentSessionKey);
                break;
            case 'appInfo':

                this.applicationState = dataJson.state;
                if (this.applicationState === ApplicationState.Init) {

                    this.droneHealth.setData(
                        DroneHealthState.Unknown, BatteryLevelWarningState.Unknown
                    );
                    this.viewStateModel.toInit();
                    CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__DRONE_HEALTH_CHECKED);
                    break;
                }

                this.droneHealth.setData(
                    dataJson.droneHealth.health, dataJson.droneHealth.batteryLevel
                );


                switch(dataJson.droneState) {
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

        this.websocket.onopen = () => {

            this.stopRetrying();

            if (startAppOnOpen && this.applicationState === ApplicationState.Started) {
                this.mainControlModel.startApp();
            }

            if (this.websocket)  {
                this.websocket.send(JSON.stringify({
                    'messageType': 'checkSessionKey'
                }));
            }

        };

        this.websocket.onclose = () => {
            this.retry();
        };
    }

    private retry() {
        clearTimeout(this.retryTimer);
        this.retryTimer = setTimeout(() => {
            this.retryCount++;
            if (STATE_CONNECTION_MAX_RETRY < this.retryCount) {
                this.stopRetrying();
                this.setModalMessage('Can not start application. Remote server may be unavailable.');        
                return;
            }
            this.setModalMessage('Trying to establish a connection to the server...');
            this.initStatesClient(false);
            this.retry();
        }, STATE_CONNECTION_RETRY_INTERVAL_MILLIS);
    }

    getDroneHealth(): DroneHealth {
        return this.droneHealth;
    }

    setTerminated(): void {
        this.applicationState = ApplicationState.Terminated;
    }

    private setModalMessage(message: string): void {
        if (this.applicationState !== ApplicationState.Terminated) {
            this.modalModel.setMessage(message);
        }
    }

    private stopRetrying(): void {
        clearTimeout(this.retryTimer);
        this.retryCount = 0;
    }

    
    private detectServerStopping(currentSessionKey: string): void {
        if (this.sessionKey !== currentSessionKey) {
            this.applicationState = ApplicationState.Terminated;
            this.stopRetrying();
            alert('The server has restarted. You have to reload the page.');
            location.href = '/';
        }
    }
}