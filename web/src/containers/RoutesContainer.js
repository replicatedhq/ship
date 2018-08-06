import { connect } from "react-redux";
import realRoutesContainer from "../components/shared/RoutesContainer";

import { getRoutes } from "../redux/data/appRoutes/actions";
import { getHelmChartMetadata } from "../redux/data/kustomizeSettings/actions";

const RoutesContainer = connect(
  state => ({
    dataLoading: state.ui.main.loading,
    routes: state.data.appRoutes.routesData.routes,
  }),
  dispatch => ({
    getHelmChartMetadata() { return dispatch(getHelmChartMetadata()) },
    getRoutes() { return dispatch(getRoutes()); },
  }),
)(realRoutesContainer);

export default RoutesContainer;