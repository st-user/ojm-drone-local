import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import ViewStateModel from './ViewStateModel';

const HEALTH_CHECK_INTERVAL = 1000;

const DRONE_HEALTH_DESCS = ['-', 'OK', 'NG'];

class DroneHealth {

    health: string;
    batteryLevel: string;

    constructor() {
        this.health = '-';
        this.batteryLevel = '-%';
    }

    setData(_health: number, _batteryLevel: number): void {
        this.health = DRONE_HEALTH_DESCS[_health] || '-';
        if (_health === 0) {
            this.batteryLevel = '-%';
        } else {
            this.batteryLevel = `${_batteryLevel}%`;
        }
        
    }
}

export default class MainControlModel {

    private readonly viewStateModel: ViewStateModel;
    private checkStarting: boolean;
    private readonly droneHealth: DroneHealth

    constructor(viewStateModel: ViewStateModel) {
        this.viewStateModel = viewStateModel;
        this.checkStarting = false;
        this.droneHealth = new DroneHealth();
    }

    async init(): Promise<void> {
        await this.droneHealthCheck();
    }

    private async droneHealthCheck(): Promise<void> {

        await fetch('/checkDroneHealth')
            .then(res => res.json())
            .then(ret => {
                const health = ret.health;
                const batteryLevel = ret.batteryLevel;

                this.droneHealth.setData(health, batteryLevel);

                CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__DRONE_HEALTH_CHECKED);
            });

        setTimeout(async () => {
            await this.droneHealthCheck();
        }, 3000);
    }

    async generateKey(startKeyConsumer: (startKey: string) => void): Promise<void> {
        await fetch('/generateKey')
            .then(res => res.json())
            .then(ret => {
                startKeyConsumer(ret.startKey);
            })
            .catch(e => {
                console.error(e);
                alert('Can not generate key. Remote server may fail to authorize me or be unavailable.');
            });
    }

    async startApp(startKey: string): Promise<void> {
        await fetch('/startApp', {
            method: 'POST',
            headers: {
                'Content-type': 'application/json'
            },
            body: JSON.stringify({
                startKey
            })
        })
            .then(res => {
                if (res.ok) {
                    return res.json();
                }
                throw new Error('Request does not success.');
            })
            .then(() => {
                this.viewStateModel.toReady();

                const wsProtocol = 0 <= location.protocol.indexOf('https') ? 'wss' : 'ws';
                const websocket = new WebSocket(`${wsProtocol}://${location.host}/state`);
                websocket.onmessage = (event: MessageEvent) => {
    
                    const dataJson = JSON.parse(event.data);
                    const messageType = dataJson.messageType;
    
                    switch(messageType) {
                    case 'stateChange':
    
                        switch(dataJson.state) {
                        case 'ready':
                            this.viewStateModel.toReady();
                            break;
    
                        case 'land':
                            this.viewStateModel.toLand();
                            break;
                        default:
                            return;
                        }
                        break;
                    default:
                        return;
                    }
                };
    
                websocket.onopen = () => {
                    console.log('open');
                };
    
    
                websocket.onclose = async () => {
                    this.viewStateModel.toReady();
                    this.startHealthCheck(startKey);
                };

            })
            .catch(e => {
                console.error(e);
                alert('Can not start signaling. Remote server may fail to validate the code or be unavailable.');
                this.viewStateModel.toInit();
            });
    }

    startHealthCheck(startKey: string): void {
        this.checkStarting = true;

        const healthCheck = async () => {
            await fetch('/healthCheck')
                .then(res => {
                    this.checkStarting = false;
                    console.log('Server become available.');
                    this.startApp(startKey);
                    return res.json();
                })
                .catch(() => {
                    this.checkStarting = true;
                    console.log('Server unavailable.');
                })
                .finally(() => {
                    if (this.checkStarting) {
                        setTimeout(healthCheck, HEALTH_CHECK_INTERVAL);
                    }
                });
               
        };
        setTimeout(healthCheck, HEALTH_CHECK_INTERVAL);

    }

    async takeoff(): Promise<void> {
        this.viewStateModel.toTakeOff();
        await fetch('/takeoff').then(res => res.json());
    }

    async land(): Promise<void> {
        this.viewStateModel.toLand();
        await fetch('/land').then(res => res.json());
    }

    getDroneHealth(): DroneHealth {
        return this.droneHealth;
    }
}