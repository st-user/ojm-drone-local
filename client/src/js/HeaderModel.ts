export default class HeaderModel {
    
    async terminate(): Promise<void> {

        let msg = 'Are you sure you want to terminate the application?';
        msg += ' If you terminate the application, the drone lands (if it has already taken off) and the server stops.';
        msg += ' In order to restart the application, you have to run the server (double click the exe file) manually.';

        if (confirm(msg)) {

            await fetch('/terminate').then(res => {
                if (res.ok) {
                    return;
                }
                throw new Error('Request does not success.');
            });
        }


    }
}