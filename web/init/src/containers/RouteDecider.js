import { connect } from "react-redux";
import realRouteDecider from "../components/shared/RouteDecider";

import { getRoutes } from "../redux/data/appRoutes/actions";
import { getMetadata } from "../redux/data/kustomizeSettings/actions";

const RouteDecider = connect(
  state => ({
    dataLoading: state.ui.main.loading,
    routes: state.data.appRoutes.routesData.routes,
    isDone: state.data.appRoutes.routesData.isDone,
  }),
  dispatch => ({
    getRoutes() { return dispatch(getRoutes()); },
    getMetadata() { return dispatch(getMetadata()) },
  }),
)(realRouteDecider);

export default RouteDecider;