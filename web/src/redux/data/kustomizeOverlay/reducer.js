import { constants } from "./actions";
import uniqBy from "lodash/uniqBy";

const kustomizeState = {
  fileContents: [],
  patch: "",
};

function updateFileContents(currState, data) {
  const nextFiles = currState.fileContents;
  const transformed = {
    baseContent: data.content.base,
    overlayContent: data.content.overlay,
    key: data.path,
    isSupported: data.content.isSupported,
  }
  nextFiles.unshift(transformed); // add to front of array so uniqBy will keep newest version
  return uniqBy(nextFiles, "key");
}

export function kustomizeData(state = kustomizeState, action) {
  switch (action.type) {
  case constants.RECEIVE_FILE_CONTENT:
    const updatedContents = updateFileContents(state, action.payload);
    return Object.assign({}, state, {
      fileContents: updatedContents
    });
  case constants.RECEIVE_PATCH:
    const { patch } = action.payload;
    return Object.assign({}, state, {
      patch,
    });
  case constants.RECEIVE_MODIFIED:
    const { modified } = action.payload;
    return Object.assign({}, state, {
      modified,
    });
  default:
    return state;
  }
}
