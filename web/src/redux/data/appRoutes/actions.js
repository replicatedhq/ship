import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
import { receiveCurrentStep } from "../determineSteps/actions";

const apiEndpoint = window.env.API_ENDPOINT;

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
  return async (dispatch) => {
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

export async function fetchContentForStep(stepId) {
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
  return async(dispatch) => {
    dispatch(polling(true));

    const intervalId = setInterval(async() => {
      const body = await fetchContentForStep(stepId).catch(() => {
        dispatch(polling(false));
        clearInterval(intervalId);
        return;
      });
      dispatch(setProgress(body.progress));

      const { progress } = body;
      const { detail } = progress;
      const parsedDetail = JSON.parse(detail);

      const finishedStatus = parsedDetail.status === "success";
      const errorStatus = parsedDetail.status === "error";

      if (finishedStatus) {
        dispatch(polling(false));
        clearInterval(intervalId);
        return cb();
      }
      if (errorStatus) {
        dispatch(polling(false));
        clearInterval(intervalId);
        return;
      }
    }, 1000);
    return;
  };
}

export function getContentForStep(stepId) {
  return async (dispatch) => {
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
