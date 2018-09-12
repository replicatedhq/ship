import { constants } from "./actions";

const loadingState = {
  appSettingsFieldsLoading: false,
  consoleSettingsFieldsLoading: false,
  saveAppSettingsLoading: false,
  getCurrentStepLoading: false,
  submitActionLoading: false,
  fileContentLoading: false,
  saveKustomizeLoading: false,
};

export function loading(state = loadingState, action = {}) {
  switch (action.type) {
  case constants.LOADING_DATA:
    return Object.assign({}, state, {
      [`${action.payload.key}Loading`]: action.payload.isLoading
    });
  default:
    return state;
  }
}
