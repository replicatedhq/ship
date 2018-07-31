import { connect } from "react-redux";
import realSidebar from "../components/shared/Sidebar";

import { loadingData } from "../redux/ui/main/actions";

const Sidebar = connect(
  state => ({
    appSettingsFieldsList: state.data.applicationSettings.settingsData.appSidebarSubItems,
  }),
  dispatch => ({
    loadingData(key, isLoading) { return dispatch(loadingData(key, isLoading)); },
  }),
)(realSidebar);

export default Sidebar;
