import { CommonEventDispatcher } from 'client-js-lib';
import { CustomEventNames } from './CustomEventNames';
import { postJsonCgi, deleteCgi } from './Auth';

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
        const errorMsg = 'Failed to update access token. The input token may be invalid.';
        await postJsonCgi('/updateAccessToken', JSON.stringify({ 'accessToken': this.accessToken }), errorMsg)
            .then(res => res.json())
            .then(ret => {
                this.accessToken = '';
                this.savedAccessTokenDesc = ret.accessTokenDesc;
                CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED);
            }).catch(e => {
                console.error(e);
                CommonEventDispatcher.dispatch(CustomEventNames.OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED);
            });
    }

    async delete(): Promise<void> {
        if (confirm('Are you sure you want to delete the existing access token?')) {
            await deleteCgi('/deleteAccessToken').then(res => res.json()).then(() => {
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