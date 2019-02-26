import { connect } from "react-redux";
import realAppWrapper from "../components/shared/AppWrapper";

import { loadingData } from "../redux/ui/main/actions";

const AppWrapper = connect(
  state => ({
    dataLoading: state.ui.main.loading,
  }),
  dispatch => ({
    loadingData(key, isLoading) { return dispatch(loadingData(key, isLoading)); },
  }),
)(realAppWrapper);

export default AppWrapper;
