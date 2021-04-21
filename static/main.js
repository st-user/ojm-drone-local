window.addEventListener('DOMContentLoaded', () => {

    const STATE = {
        INIT: 0,
        READY: 1,
        LAND: 2,
        TAKEOFF: 3
    };

    let websocket;
    let checkStarting;
    let state;
    const HEALTH_CHECK_INTERVAL = 1000;

    const _q = selector => document.querySelector(selector);
    const _click = ($elem, handler) => $elem.addEventListener('click', handler);
        
    const $startKey = _q('#startKey');
    const $generateKey = _q('#generateKey');
    const $start = _q('#start');

    const $takeoff = _q('#takeoff');
    const $land = _q('#land');


    function init() {
        state = STATE.INIT;
        initView();
    }

    function initView() {
        $startKey.disabled = false;
        enableStartButtons();
        disableControlButtons();
    }

    function ready() {
        state = STATE.READY;
        readyView();
    }

    function readyView() {
        $startKey.disabled = true;
        disableStartButtons();
        disableControlButtons();
    }

    function land() {
        state = STATE.LAND;
        landView();
    }

    function landView() {
        $startKey.disabled = true;
        disableStartButtons();
        enableControlButtons();
    }

    function takeoff() {
        state = STATE.TAKEOFF;
        takeoffView();
    }

    function takeoffView() {
        $startKey.disabled = true;
        disableStartButtons();
        enableControlButtons();
    }

    function resetClass($elem, classToAdd, classToRemove) {
        $elem.classList.remove(classToRemove);
        $elem.classList.add(classToAdd);        
    }

    function disableElem($elem) {
        resetClass($elem, 'disabled', 'enabled');
    }

    function enableElem($elem) {
        resetClass($elem, 'enabled', 'disabled');
    }

    function disableStartButtons() {
        disableElem($start);
        disableElem($generateKey);
    }

    function enableStartButtons() {
        enableElem($start);
        enableElem($generateKey);
    }

    function disableControlButtons() {
        disableElem($takeoff);
        disableElem($land);
    }

    function enableControlButtons() {
        enableElem($takeoff);
        enableElem($land);
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
                ready();

                const wsProtocol = 0 <= location.protocol.indexOf('https') ? 'wss' : 'ws';
                websocket = new WebSocket(`${wsProtocol}://${location.host}/state`);
                websocket.onmessage = event => {
    
                    const dataJson = JSON.parse(event.data);
                    const messageType = dataJson.messageType;
    
                    switch(messageType) {
                    case 'stateChange':
    
                        switch(dataJson.state) {
                        case 'ready':
                            ready();
                            break;
    
                        case 'land':
                            land();
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
                    readyView();
                    await startHealthCheck();
                };

            })
            .catch(e => {
                console.error(e);
            });


    }

    _click($generateKey, async () => {
        if (state !== STATE.INIT) {
            return;
        }

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

    _click($start, async () => {
        if (state !== STATE.INIT) {
            return;
        }
        await startApp();
    });

    _click($takeoff, async () => {
        if (state !== STATE.LAND && state !== STATE.TAKEOFF) {
            return;
        }
        await fetch('/takeoff').then(res => res.json());
        takeoff();
    });

    _click($land, async () => {
        if (state !== STATE.LAND && state !== STATE.TAKEOFF) {
            return;
        }
        await fetch('/land').then(res => res.json());
    });

    init();

});