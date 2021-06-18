import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';

export default class SetupModel {
 
    
    private accessToken: string;
    private savedAccessTokenDesc: string;

    constructor() {
        this.accessToken = '';
        this.savedAccessTokenDesc = '';
    }

    async update(): Promise<void> {
        if (this.getSavedAccessTokenDesc()) {
            if (!confirm('Are you sure you want to update the existing access token?')) {
                return;
            }
        }
        await fetch('/updateAccessToken', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json'
            },
            body: JSON.stringify({
                'accessToken': this.accessToken
            })
        }).then(res => {
            if (res.status !== 200) {
                throw new Error('Invalid response');
            }
            return res.json();
        }).then(ret => {
            this.accessToken = '';
            this.savedAccessTokenDesc = ret.accessTokenDesc;
            CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED);
        }).catch(() => {
            alert('Failed to update access token. The input token may be invalid.');
            CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED);
        });
    }

    async delete(): Promise<void> {
        if (confirm('Are you sure you want to delete the existing access token?')) {
            await fetch('/deleteAccessToken', {
                method: 'DELETE'
            }).then(res => res.json()).finally(() => {
                this.accessToken = '';
                this.savedAccessTokenDesc = '';
                CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED);
            });
        }
    }

    setAccessToken(accessToken: string): void {
        this.accessToken = accessToken;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED);
    }

    setSavedAccessTokenDesc(accessTokenDesc: string): void {
        this.savedAccessTokenDesc = accessTokenDesc;
        CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED);
    }

    getAccessToken(): string {
        return this.accessToken;
    }

    getSavedAccessTokenDesc(): string {
        return this.savedAccessTokenDesc;
    }

    canUpdate(): boolean {
        return this.accessToken.length > 0;
    }

    canDelete(): boolean {
        return this.savedAccessTokenDesc.length > 0;
    }
}