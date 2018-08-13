import { combineReducers } from "redux";
import appRoutes from "./appRoutes";
import applicationSettings from "./applicationSettings";
import determineSteps from "./determineSteps";
import channelSettings from "./channelSettings";
import kustomizeOverlay from "./kustomizeOverlay";
import kustomizeSettings from "./kustomizeSettings";

export default combineReducers({
  appRoutes,
  applicationSettings,
  determineSteps,
  channelSettings,
  kustomizeOverlay,
  kustomizeSettings
});
