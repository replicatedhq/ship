import { constants } from "./actions";

const shipAppMetadataState = {
  name: "",
  version: "",
  release: "",
  icon: "",
  description: "",
  readme: "",
  values: "",
  error: false,
  errorMessage: "",
  loaded: false,
};

export function shipAppMetadata(state = shipAppMetadataState, action) {
  switch (action.type) {
  case constants.RECEIVE_METADATA:
    return Object.assign({}, state, {
      ...action.payload,
      loaded: true,
    });
  case constants.SET_HELM_CHART_ERROR:
    return Object.assign({}, state, {
      error: true,
      errorMessage: action.payload.error
    });
  default:
    return state;
  }
}
