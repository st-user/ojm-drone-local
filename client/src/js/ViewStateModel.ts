import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';


enum State {
    INIT,
    READY,
    LAND,
    TAKEOFF
}

export default class ViewStateModel {

    private state: State;

    constructor() {
        this.state = State.INIT;
    }

    isInit(): boolean {
        return this.is(State.INIT);
    }

    toInit(): void {
        this.setState(State.INIT);
    }

    isReady(): boolean {
        return this.is(State.READY);
    }

    toReady(): void {
        this.setState(State.READY);
    }

    isLand(): boolean {
        return this.is(State.LAND);
    }

    toLand(): void {
        this.setState(State.LAND);
    }

    isTakeOff(): boolean {
        return this.is(State.TAKEOFF);
    }

    toTakeOff(): void {
        this.setState(State.TAKEOFF);
    }

    private is(value: State): boolean {
        return this.state === value;
    }

    private setState(value: State): void {
        this.state = value;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__VIEW_STATE_CHANGED);
    }
}