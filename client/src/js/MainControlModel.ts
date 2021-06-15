import ViewStateModel from './ViewStateModel';

const HEALTH_CHECK_INTERVAL = 1000;

export default class MainControlModel {

    private readonly viewStateModel: ViewStateModel;
    private checkStarting: boolean;   

    constructor(viewStateModel: ViewStateModel) {
        this.viewStateModel = viewStateModel;
        this.checkStarting = false;
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

}