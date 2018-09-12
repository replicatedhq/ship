import { constants } from "./actions";

const channelSettingsState = {
  channel: {},
};

export function channelSettingsData(state = channelSettingsState, action) {
  switch (action.type) {
  case constants.RECEIVE_CHANNEL_DETAILS:
    return Object.assign({}, state, {
      channel: action.payload,
    });
  default:
    return state;
  }
}
