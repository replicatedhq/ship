import { connect } from "react-redux";
import realDetermineComponentForRoute from "../components/shared/DetermineComponentForRoute";

import { getChannel } from "../redux/data/channelSettings/actions";
import { getCurrentStep, submitAction } from "../redux/data/determineSteps/actions";
import { getHelmChartMetadata, saveHelmChartValues } from "../redux/data/kustomizeSettings/actions";

const DetermineComponentForRoute = connect(
  state => ({
    dataLoading: state.ui.main.loading,
    currentStep: state.data.determineSteps.stepsData.step,
    helmChartMetadata: state.data.kustomizeSettings.helmChartMetadata,
    phase: state.data.determineSteps.stepsData.phase,
    actions: state.data.determineSteps.stepsData.actions,
    progress: state.data.determineSteps.stepsData.progress,
  }),
  dispatch => ({
    getChannel() { return dispatch(getChannel()); },
    getCurrentStep(loaderType, stepId) { return dispatch(getCurrentStep(loaderType, stepId)); },
    getHelmChartMetadata() { return dispatch(getHelmChartMetadata()) },
    saveHelmChartValues(payload) { return dispatch(saveHelmChartValues(payload)) },
    submitAction(action) { return dispatch(submitAction(action)); }
  }),
)(realDetermineComponentForRoute);

export default DetermineComponentForRoute;