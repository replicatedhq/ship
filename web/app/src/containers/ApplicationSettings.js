import { connect } from "react-redux";
import realApplicationSettings from "../components/application_settings/ApplicationSettings";

import {
  getApplicationSettings,
  saveApplicationSettings,
  finalizeApplicationSettings,
  setApplicationState
} from "../redux/data/applicationSettings/actions";

const ApplicationSettings = connect(
  state => ({
    settingsFields: state.data.applicationSettings.settingsData.settingsFields,
    settingsFieldsList: state.data.applicationSettings.settingsData.settingsFieldsList,
    dataLoading: state.ui.main.loading,
  }),
  dispatch => ({
    getApplicationSettings(payload, shouldLoad) { return dispatch(getApplicationSettings(payload, shouldLoad)); },
    saveApplicationSettings(payload, validate) { return dispatch(saveApplicationSettings(payload, validate)); },
    finalizeApplicationSettings(payload, validate) { return dispatch(finalizeApplicationSettings(payload, validate)); },
    setApplicationState(payload) { return dispatch(setApplicationState(payload)); }
  }),
)(realApplicationSettings);

export default ApplicationSettings;
