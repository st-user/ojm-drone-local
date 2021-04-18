window.addEventListener('DOMContentLoaded', () => {

        let websocket;
        let checkStarting;
        const HEALTH_CHECK_INTERVAL = 1000;

        const _q = selector => document.querySelector(selector);
        const _click = ($elem, handler) => $elem.addEventListener('click', handler);
        
        const $startKey = _q('#startKey');
        const $generateKey = _q('#generateKey');
        const $start = _q('#start');

        const $takeoff = _q('#takeoff');
        const $land = _q('#land');


        function init() {
            initView();
        }

        function initView() {
            $startKey.disabled = false;
            $generateKey.disabled = false;
            $start.disabled = false;

            $takeoff.disabled = true;
            $land.disabled = true;
        }

        function ready() {
            readyView();
        }

        function readyView() {
            $startKey.disabled = true;
            $generateKey.disabled = true;
            $start.disabled = true;

            $takeoff.disabled = true;
            $land.disabled = true;
        }

        function land() {
            landView();
        }

        function landView() {
            $startKey.disabled = true;
            $generateKey.disabled = true;
            $start.disabled = true;

            $takeoff.disabled = false;
            $land.disabled = false;
        }

        function takeoff() {
            takeoffView();
        }

        function takeoffView() {
            $startKey.disabled = true;
            $generateKey.disabled = true;
            $start.disabled = true;

            $takeoff.disabled = false;
            $land.disabled = false;
        }

        async function startHealthCheck() {
            checkStarting = true;

            async function healthCheck() {
                await fetch('/healthCheck')
                    .then(async res => {
                        checkStarting = false;
                        console.log('Server become available.');
                        await startApp();
                        return res.json();
                    })
                    .catch(() => {
                        checkStarting = true;
                        console.log('Server unavailable.');
                    })
                    .finally(() => {
                        if (checkStarting) {
                            setTimeout(healthCheck, HEALTH_CHECK_INTERVAL);
                        }
                    });
               
            }
            setTimeout(healthCheck, HEALTH_CHECK_INTERVAL);

        }

        async function startApp() {

            await fetch('/startApp', {
                method: 'POST',
                headers: {
                    'Content-type': 'application/json'
                },
                body: JSON.stringify({
                    startKey: $startKey.value
                })
            })
            .then(res => res.json())
            .then(() => {
                ready()

                const wsProtocol = 0 <= location.protocol.indexOf('https') ? 'wss' : 'ws';
                websocket = new WebSocket(`${wsProtocol}://${location.host}/state`);
                websocket.onmessage = event => {
    
                    const dataJson = JSON.parse(event.data);
                    const messageType = dataJson.messageType;
    
                    switch(messageType) {
                    case 'stateChange':
    
                        switch(dataJson.state) {
                        case 'ready':
                            ready()
                            break;
    
                        case 'land':
                            land()
                            break;
                        default:
                            return;
                        }
    
                    default:
                        return;
                    }
                };
    
                websocket.onopen = event => {
                    console.log('open');
                };
    
    
                websocket.onclose = async event => {
                    readyView();
                    await startHealthCheck();
                };

            })
            .catch(e => {
                console.error(e);
            });


        }

        _click($generateKey, async () => {

            await fetch('/generateKey')
                    .then(res => res.json())
                    .then(ret => {
                        $startKey.value = ret.startKey;
                    })
                    .catch(e => {
                        console.error(e);
                        alert('Can not generate key. Remote server may fail to authorize me or be unavailable.');
                    });


        });

        _click($start, startApp);

        _click($takeoff, async () => {
            await fetch('/takeoff').then(res => res.json());
        });

        _click($land, async () => {
            await fetch('/land').then(res => res.json());
        });

        init();

});