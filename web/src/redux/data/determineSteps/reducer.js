import { constants } from "./actions";

const stepsDataState = {
  step: {},
  phase: "loading",
  progress: {},
  stepError: "",
  actions: [],
  kustomizeFlow: false
};

export function stepsData(state = stepsDataState, action) {
  switch (action.type) {
  case constants.RECEIVE_CURRENT_STEP:
    const { currentStep } =  action.payload;
    const isKustomize = currentStep.helmIntro || currentStep.helmValues || currentStep.kustomize;
    return Object.assign({}, state, {
      step: currentStep,
      phase: action.payload.phase,
      progress: action.payload.progress,
      actions: action.payload.actions,
      kustomizeFlow: isKustomize
    });
  case constants.SET_STEP_ERROR:
    return Object.assign({}, state, {
      stepError: action.payload
    });
  default:
    return state;
  }
}
