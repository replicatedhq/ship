import { connect } from "react-redux";
import realKustomizeOverlay from "../components/kustomize/kustomize_overlay/KustomizeOverlay";

import { loadingData } from "../redux/ui/main/actions";
import { getCurrentStep } from "../redux/data/determineSteps/actions";
import {
  getFileContent,
  saveKustomizeOverlay,
  finalizeKustomizeOverlay,
} from "../redux/data/kustomizeOverlay/actions";

const KustomizeOverlay = connect(
  state => ({
    currentStep: state.data.determineSteps.stepsData.step,
    phase: state.data.determineSteps.stepsData.phase,
    actions: state.data.determineSteps.stepsData.actions,
    progress: state.data.determineSteps.stepsData.progress,
    fileContents: state.data.kustomizeOverlay.kustomizeData.fileContents,
    dataLoading: state.ui.main.loading,
  }),
  dispatch => ({
    getCurrentStep(loaderType) { return dispatch(getCurrentStep(loaderType)); },
    getFileContent(payload) { return dispatch(getFileContent(payload)); },
    saveKustomizeOverlay(payload) { return dispatch(saveKustomizeOverlay(payload)); },
    finalizeKustomizeOverlay() { return dispatch(finalizeKustomizeOverlay()); },
    loadingData(key, isLoading) { return dispatch(loadingData(key, isLoading)); },
  }),
)(realKustomizeOverlay);

export default KustomizeOverlay;
