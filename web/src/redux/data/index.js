import { combineReducers } from "redux";
import applicationSettings from "./applicationSettings";
import determineSteps from "./determineSteps";
import channelSettings from "./channelSettings";
import kustomizeOverlay from "./kustomizeOverlay";
import kustomizeSettings from "./kustomizeSettings";

export default combineReducers({
  applicationSettings,
  determineSteps,
  channelSettings,
  kustomizeOverlay,
  kustomizeSettings
});
