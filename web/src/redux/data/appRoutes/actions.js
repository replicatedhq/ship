import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
import { receiveCurrentStep } from "../determineSteps/actions";

const apiEndpoint = window.env.LIFECYCLE_ENDPOINT;

export const constants = {
  RECEIVE_ROUTES: "RECEIVE_ROUTES",
  SET_PHASE: "SET_PHASE"
};

export function receiveRoutes(routes) {
  return {
    type: constants.RECEIVE_ROUTES,
    payload: routes
  };
}

export function setPhase(phase) {
  return {
    type: constants.SET_PHASE,
    payload: phase
  }
}

export function getRoutes() {
  return async (dispatch) => {
    let response;
    dispatch(loadingData("routes", true));
    try {
      const url = `${apiEndpoint}/lifecycle`;
      response = await fetch(url, {
        method: "GET",
        headers: {
          "Accept": "application/json",
        },
      });
      if (!response.ok) {
        dispatch(loadingData("routes", false));
        return;
      }
      const body = await response.json();
      dispatch(loadingData("routes", false));
      dispatch(receiveRoutes(body));
    } catch (error) {
      console.log(error);
      return;
    }
  };
}

export function getContentForStep(stepId) {
  return async (dispatch) => {
    let response;
    dispatch(loadingData("getCurrentStep", true));
    try {
      const url = `${apiEndpoint}/lifecycle/step/${stepId}`;
      response = await fetch(url, {
        method: "GET",
        headers: {
          "Accept": "application/json",
        },
      });
      if (!response.ok) {
        dispatch(loadingData("getCurrentStep", false));
        if (response.status === 400) {
          const body = await response.json();
          if (body) {
            dispatch(receiveCurrentStep(body));
          }
        }
        return;
      }
      const body = await response.json();
      dispatch(loadingData("getCurrentStep", false));
      dispatch(receiveCurrentStep(body));
    } catch (error) {
      console.log(error);
      return;
    }
  };
}

export function finalizeStep(payload) {
  const { uri, method, body } = payload.action.onclick;
  return async (dispatch) => {
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
      // dispatch(getCurrentStep("getCurrentStep", nextStep));
    } catch (error) {
      console.log(error);
      dispatch(loadingData("postConfirm", false));
      return;
    }
  };
}
