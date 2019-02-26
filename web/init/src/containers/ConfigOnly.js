import { connect } from "react-redux";
import realConfigOnly from "../components/config_only/ConfigOnly";

import {
  getApplicationSettings,
  saveApplicationSettings,
  finalizeApplicationSettings,
  setApplicationState
} from "../redux/data/applicationSettings/actions";
import { getCurrentStep } from "../redux/data/determineSteps/actions";

const ConfigOnly = connect(
  state => ({
    settingsFields: state.data.applicationSettings.settingsData.settingsFields,
    settingsFieldsList: state.data.applicationSettings.settingsData.settingsFieldsList,
    configErrors: state.data.applicationSettings.settingsData.configErrors,
    phase: state.data.determineSteps.stepsData.phase,
    dataLoading: state.ui.main.loading,
  }),
  dispatch => ({
    getCurrentStep() { return dispatch(getCurrentStep()); },
    getApplicationSettings(payload, shouldLoad) { return dispatch(getApplicationSettings(payload, shouldLoad)); },
    saveApplicationSettings(payload, validate) { return dispatch(saveApplicationSettings(payload, validate)); },
    finalizeApplicationSettings(payload, validate) { return dispatch(finalizeApplicationSettings(payload, validate)); },
    setApplicationState(payload) { return dispatch(setApplicationState(payload)); }
  }),
)(realConfigOnly);

export default ConfigOnly;
