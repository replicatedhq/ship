import { connect } from "react-redux";
import realNavBar from "../components/shared/NavBar";

import { loadingData } from "../redux/ui/main/actions";

const NavBar = connect(
  state => ({
    channelDetails: state.data.channelSettings.channelSettingsData.channel,
    helmChartMetadata: state.data.kustomizeSettings.helmChartMetadata,
    phase: state.data.determineSteps.stepsData.phase,
    isKustomizeFlow: state.data.appRoutes.routesData.isKustomizeFlow,
    isKustomize: state.data.determineSteps.stepsData.kustomizeFlow,
    dataLoading: state.ui.main.loading,
  }),
  dispatch => ({
    loadingData(key, isLoading) { return dispatch(loadingData(key, isLoading)); },
  }),
)(realNavBar);

export default NavBar;
