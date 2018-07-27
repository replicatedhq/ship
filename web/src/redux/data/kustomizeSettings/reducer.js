import { constants } from "./actions";

const helmChartMetadataState = {
  name: "",
  version: "",
  release: "",
  icon: "",
  description: "",
  readme: "",
  values: "",
  error: false,
  errorMessage: ""
};

export function helmChartMetadata(state = helmChartMetadataState, action) {
  switch (action.type) {
  case constants.RECEIVE_HELM_CHART_METADATA:
    return Object.assign({}, state, {
      name: action.payload.metadata.name,
      version: action.payload.metadata.version,
      release: action.payload.metadata.release,
      icon: action.payload.metadata.icon,
      description: action.payload.metadata.description,
      readme: action.payload.metadata.readme,
      values: action.payload.metadata.values,
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
