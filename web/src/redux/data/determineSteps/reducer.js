import { constants } from "./actions";

const stepsDataState = {
  step: {},
  phase: "loading",
  progress: {},
  stepError: "",
  actions: [],
};

export function stepsData(state = stepsDataState, action) {
  switch (action.type) {
  case constants.RECEIVE_CURRENT_STEP:
    return Object.assign({}, state, {
      step: action.payload.currentStep,
      phase: action.payload.phase,
      progress: action.payload.progress,
      actions: action.payload.actions
    });
  case constants.SET_STEP_ERROR:
    return Object.assign({}, state, {
      stepError: action.payload
    });
  default:
    return state;
  }
}
