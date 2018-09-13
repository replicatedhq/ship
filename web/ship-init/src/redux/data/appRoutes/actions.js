import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
import { receiveCurrentStep } from "../determineSteps/actions";

export const constants = {
  RECEIVE_ROUTES: "RECEIVE_ROUTES",
  SET_PHASE: "SET_PHASE",
  SET_PROGRESS: "SET_PROGRESS",
  POLLING: "POLLING",
  SHUTDOWN_APP: "SHUTDOWN_APP",
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

export function polling(isPolling) {
  return {
    type: constants.POLLING,
    payload: isPolling,
  };
}

export function setProgress(progress) {
  return {
    type: constants.SET_PROGRESS,
    payload: progress,
  };
}

export function shutdownApp() {
  return {
    type: constants.SHUTDOWN_APP
  }
}

export function getRoutes() {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    let response;
    dispatch(loadingData("routes", true));
    try {
      const url = `${apiEndpoint}/navcycle`;
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

export async function fetchContentForStep(apiEndpoint, stepId) {
  const url = `${apiEndpoint}/navcycle/step/${stepId}`;
  const response = await fetch(url, {
    method: "GET",
    headers: {
      "Accept": "application/json",
    },
  });
  const body = await response.json();
  return body;
}

export function pollContentForStep(stepId, cb) {
  return async(dispatch, getState) => {
    dispatch(polling(true));

    const { apiEndpoint } = getState();
    const intervalId = setInterval(async() => {
      const body = await fetchContentForStep(apiEndpoint, stepId).catch(() => {
        dispatch(polling(false));
        clearInterval(intervalId);
        return;
      });
      dispatch(setProgress(body.progress));

      const { progress } = body;
      const { detail } = progress;
      const { status: parsedDetailStatus } = JSON.parse(detail);

      const finishedStatus = parsedDetailStatus === "success";
      const messageStatus = parsedDetailStatus === "message";
      const errorStatus = parsedDetailStatus === "error";

      if (finishedStatus) {
        dispatch(polling(false));
        clearInterval(intervalId);
        return cb();
      }
      if (errorStatus || messageStatus) {
        dispatch(polling(false));
        clearInterval(intervalId);
        return;
      }
    }, 1000);
    return;
  };
}

export function getContentForStep(stepId) {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();

    let response;
    dispatch(loadingData("getCurrentStep", true));
    try {
      const url = `${apiEndpoint}/navcycle/step/${stepId}`;
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
      let resp = body;
      if(!body.currentStep) {
        resp["currentStep"] = {}
      }
      dispatch(receiveCurrentStep(resp));
    } catch (error) {
      console.log(error);
      return;
    }
  };
}

export function finalizeStep(payload) {
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
      // dispatch(getCurrentStep("getCurrentStep", nextStep));
    } catch (error) {
      console.log(error);
      dispatch(loadingData("postConfirm", false));
      return;
    }
  };
}

export function initializeStep(stepId) {
  return async(dispatch, getState) => {
    const { apiEndpoint } = getState();
    try {
      const url = `${apiEndpoint}/navcycle/step/${stepId}`;
      await fetch(url, {
        method: "POST",
        headers: {
          "Accept": "application/json",
        },
      });
    } catch (error) {
      console.log(error);
      return;
    }
  };
}
