import { postJsonCgi } from './Auth';
import Messages from './Messages';

export default class HeaderModel {
    
    async terminate(): Promise<boolean> {
        if (confirm(Messages.msg.HeaderModel_001)) {
            await postJsonCgi('/terminate');
            return true;
        }
        return false;
    }
}