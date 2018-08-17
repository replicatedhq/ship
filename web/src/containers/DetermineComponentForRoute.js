import { connect } from "react-redux";
import realDetermineComponentForRoute from "../components/shared/DetermineComponentForRoute";

import { getChannel } from "../redux/data/channelSettings/actions";
import { getContentForStep, pollContentForStep, finalizeStep } from "../redux/data/appRoutes/actions";
import { getHelmChartMetadata, saveHelmChartValues } from "../redux/data/kustomizeSettings/actions";

const DetermineComponentForRoute = connect(
  state => ({
    dataLoading: state.ui.main.loading,
    currentStep: state.data.determineSteps.stepsData.step,
    helmChartMetadata: state.data.kustomizeSettings.helmChartMetadata,
    actions: state.data.determineSteps.stepsData.actions,
    phase: state.data.determineSteps.stepsData.phase,
    progress: state.data.determineSteps.stepsData.progress,
    isNewRouter: state.data.appRoutes.routesData.isKustomizeFlow,
    isPolling: state.data.determineSteps.stepsData.isPolling,
  }),
  dispatch => ({
    getChannel() { return dispatch(getChannel()); },
    getContentForStep(stepId) { return dispatch(getContentForStep(stepId)); },
    pollContentForStep(stepId, cb) { return dispatch(pollContentForStep(stepId, cb)); },
    getHelmChartMetadata() { return dispatch(getHelmChartMetadata()) },
    saveHelmChartValues(payload) { return dispatch(saveHelmChartValues(payload)) },
    finalizeStep(action) { return dispatch(finalizeStep(action)); }
  }),
)(realDetermineComponentForRoute);

export default DetermineComponentForRoute;