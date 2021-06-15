import { CustomEventNamesFactory } from 'client-js-lib';

const CustomEventNames = CustomEventNamesFactory.createNames();
const CustomEventContextNames = CustomEventNamesFactory.createNames();

CustomEventNames
    .set('OJM_DRONE_LOCAL__VIEW_STATE_CHANGED', 'ojm-drone-local/view-state-changed')

;

export { CustomEventNames, CustomEventContextNames };
