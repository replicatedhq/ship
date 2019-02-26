import { combineReducers } from "redux";
import appRoutes from "./appRoutes";
import applicationSettings from "./applicationSettings";
import determineSteps from "./determineSteps";
import kustomizeOverlay from "./kustomizeOverlay";
import kustomizeSettings from "./kustomizeSettings";

export default combineReducers({
  appRoutes,
  applicationSettings,
  determineSteps,
  kustomizeOverlay,
  kustomizeSettings
});
