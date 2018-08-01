import { constants } from "./actions";

const kustomizeState = {
  fileContents: [],
  patch: "",
};

function updateFileContents(currState, data) {
  const nextFiles = currState.fileContents;
  let newObj = {};
  newObj.baseContent = data.content.base;
  newObj.overlayContent = data.content.overlay;
  newObj.key = data.path;
  nextFiles.push(newObj);
  return nextFiles;
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
    return {
      ...state,
      patch,
    }
  default:
    return state;
  }
}
