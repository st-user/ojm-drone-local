enum CustomEventNames {
    OJM_DRONE_LOCAL__VIEW_STATE_CHANGED = 'ojm-drone-local/view-state-changed',
    OJM_DRONE_LOCAL__TAB_CLICKED = 'ojm-drone-local/tab-clicked',
    OJM_DRONE_LOCAL__ACCESS_TOKEN_INPUT_STATE_CHANGED = 'ojm-drone-local/access-token-input-state-changed',
    OJM_DRONE_LOCAL__PROGRESS_STATE_CHANGED = 'ojm-drone-local/progress-state-changed',
    OJM_DRONE_LOCAL__START_KEY_INPUT_STATE_CHANGED = 'ojm-drone-local/start-key-input-state-changed',
    OJM_DRONE_LOCAL__DRONE_HEALTH_CHECKED = 'ojm-drone-local/drone-health-checked',
    OJM_DRONE_LOCAL__TOGGLE_MODAL_MESSAGE = 'ojm-drone-local/toggle-modal-message',
    OJM_DRONE_LOCAL__SESSION_KEY_SUCCESSFULLY_RETRIVED = 'ojm-drone-local/session-key-successfully-retrived',
    OJM_DRONE_LOCAL__SESSION_KEY_AUTHORIZED_ACCESS_ENABLED = 'ojm-drone-local/authorized-access-enabled',
}

enum CustomEventContextNames {}

export { CustomEventNames, CustomEventContextNames };
