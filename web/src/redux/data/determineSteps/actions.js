import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
import { Utilities } from "../../../utilities/utilities";

export const constants = {
  RECEIVE_CURRENT_STEP: "RECEIVE_CURRENT_STEP",
  SET_STEP_ERROR: "SET_STEP_ERROR"
};

export function receiveCurrentStep(step) {
  return {
    type: constants.RECEIVE_CURRENT_STEP,
    payload: step
  };
}

export function setStepError(message) {
  return {
    type: constants.SET_STEP_ERROR,
    payload: message
  }
}

export function getCurrentStep(loaderType = "getCurrentStep") {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    let response;
    dispatch(loadingData(loaderType, true));
    try {
      const url = `${apiEndpoint}/lifecycle/current`;
      response = await fetch(url, {
        method: "GET",
        headers: {
          "Accept": "application/json",
        },
      });
      if (!response.ok) {
        dispatch(loadingData(loaderType, false));
        return;
      }
      const body = await response.json();
      dispatch(loadingData(loaderType, false));
      dispatch(receiveCurrentStep(body));
    } catch (error) {
      console.log(error);
      if (Utilities.isFailedToFetchErr(error.message)) {
        dispatch(receiveCurrentStep({ currentStep: {}, phase: "loading"}));
      } else {
        dispatch(setStepError(error.message));
      }
      return;
    }
  };
}

export function submitAction(payload) {
  const { uri, method, body } = payload.action.onclick;
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    let response;
    dispatch(loadingData("submitAction", true));
    try {
      const url = `${apiEndpoint}${uri}`;
      response = await fetch(url, {
        method,
        body,
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        },
      });
      if (!response.ok) {
        dispatch(loadingData("submitAction", false));
      }
      dispatch(loadingData("submitAction", false));
      dispatch(getCurrentStep());
    } catch (error) {
      console.log(error);
      dispatch(loadingData("postConfirm", false));
      return;
    }
  };
}
