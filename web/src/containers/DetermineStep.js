import { connect } from "react-redux";
import realDetermineStep from "../components/shared/DetermineStep";

import { getChannel } from "../redux/data/channelSettings/actions";
import { getCurrentStep, submitAction } from "../redux/data/determineSteps/actions";
import { getHelmChartMetadata, saveHelmChartValues } from "../redux/data/kustomizeSettings/actions";

const DetermineStep = connect(
  state => ({
    dataLoading: state.ui.main.loading,
    currentStep: state.data.determineSteps.stepsData.step,
    shipAppMetadata: state.data.kustomizeSettings.shipAppMetadata,
    phase: state.data.determineSteps.stepsData.phase,
    actions: state.data.determineSteps.stepsData.actions,
    progress: state.data.determineSteps.stepsData.progress,
  }),
  dispatch => ({
    getChannel() { return dispatch(getChannel()); },
    getCurrentStep(loaderType) { return dispatch(getCurrentStep(loaderType)); },
    getHelmChartMetadata() { return dispatch(getHelmChartMetadata()) },
    saveHelmChartValues(payload) { return dispatch(saveHelmChartValues(payload)) },
    submitAction(action) { return dispatch(submitAction(action)); }
  }),
)(realDetermineStep);

export default DetermineStep;
