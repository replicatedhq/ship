import { connect } from "react-redux";
import realKustomizeOverlay from "../components/kustomize/kustomize_overlay/KustomizeOverlay";

import { loadingData } from "../redux/ui/main/actions";
import { getContentForStep } from "../redux/data/appRoutes/actions";
import { getHelmChartMetadata } from "../redux/data/kustomizeSettings/actions";
import {
  getFileContent,
  saveKustomizeOverlay,
  deleteOverlay,
  includeBase,
  finalizeKustomizeOverlay,
  generatePatch,
  applyPatch,
} from "../redux/data/kustomizeOverlay/actions";

const KustomizeOverlay = connect(
  state => ({
    currentStep: state.data.determineSteps.stepsData.step,
    phase: state.data.determineSteps.stepsData.phase,
    actions: state.data.determineSteps.stepsData.actions,
    progress: state.data.determineSteps.stepsData.progress,
    fileContents: state.data.kustomizeOverlay.kustomizeData.fileContents,
    dataLoading: state.ui.main.loading,
    patch: state.data.kustomizeOverlay.kustomizeData.patch,
    modified: state.data.kustomizeOverlay.kustomizeData.modified,
  }),
  dispatch => ({
    getFileContent(payload) { return dispatch(getFileContent(payload)); },
    getContentForStep(stepId) { return dispatch(getContentForStep(stepId)); },
    getHelmChartMetadata() { return dispatch(getHelmChartMetadata()) },
    saveKustomizeOverlay(payload) { return dispatch(saveKustomizeOverlay(payload)); },
    deleteOverlay(path, type) { return dispatch(deleteOverlay(path, type)); },
    includeBase(path) { return dispatch(includeBase(path)); },
    finalizeKustomizeOverlay() { return dispatch(finalizeKustomizeOverlay()); },
    loadingData(key, isLoading) { return dispatch(loadingData(key, isLoading)); },
    generatePatch(payload) { return dispatch(generatePatch(payload)); },
    applyPatch(payload) { return dispatch(applyPatch(payload)); },
  }),
)(realKustomizeOverlay);

export default KustomizeOverlay;
