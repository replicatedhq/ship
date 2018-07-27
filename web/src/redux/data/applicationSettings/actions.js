import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
import { Utilities } from "../../../utilities/utilities";

const apiEndpoint = window.env.API_ENDPOINT;
export const constants = {
  RECEIVE_APPLICATION_SETTINGS: "RECEIVE_APPLICATION_SETTINGS",
  SET_CONFIG_ERRORS: "SET_CONFIG_ERRORS"
};

export function receiveApplicationSettings(fields) {
  return {
    type: constants.RECEIVE_APPLICATION_SETTINGS,
    payload: fields
  };
}

export function setConfigErrors(error) {
  return {
    type: constants.SET_CONFIG_ERRORS,
    payload: error
  }
}

export function getApplicationSettings(payload, shouldLoad = true) {
  return async (dispatch) => {
    // if (!appId) return;
    let response;
    if (shouldLoad) {
      dispatch(loadingData("appSettingsFields", true));
    }
    try {
      const url = `${apiEndpoint}/config/live`;
      response = await fetch(url, {
        method: "POST",
        body: JSON.stringify(payload),
        headers: {
          "Content-Type": "application/json"
        },
      });
      if (!response.ok) {
        if (response.status === 401) {
          // unauthorized
        }
        dispatch(loadingData("appSettingsFields", false));
        throw new Error(`Unexpected response status code ${response.status}`);
      }
      const body = await response.json();
      dispatch(loadingData("appSettingsFields", false));
      dispatch(receiveApplicationSettings(body));
      return body;
    } catch (error) {
      console.log(error);
      dispatch(loadingData("appSettingsFields", false));
      if (Utilities.isFailedToFetchErr(error.message)) {
        window.location.href = "/";
      }
      return;
    }
  };
}

export function saveApplicationSettings(payload, validate) {
  return async (dispatch) => {
    dispatch(loadingData("saveAppSettings", true));
    let response;
    try {
      const url = `${apiEndpoint}/config`;
      response = await fetch(url, {
        method: "PUT",
        body: JSON.stringify({
          options: payload,
          validate,
        }),
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        }
      });
      if (!response.ok) {
        if (response.status === 401) {
          // unauthorized
        }
        const serverErr = await response.json();
        dispatch(setConfigErrors(serverErr));
        throw new Error(`Unexpected response status code ${response.status}`);
      }
      const body = await response.json();
      dispatch(loadingData("saveAppSettings", false));
      return body;
    } catch (error) {
      dispatch(loadingData("saveAppSettings", false));
      console.log(error);
      return false;
    }
  }
}

export function finalizeApplicationSettings(payload, validate) {
  return async (dispatch) => {
    dispatch(loadingData("finalizeAppSettings", true));
    let response;
    try {
      const url = `${apiEndpoint}/config/finalize`;
      response = await fetch(url, {
        method: "PUT",
        body: JSON.stringify({
          options: payload,
          validate,
        }),
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        }
      });
      if (!response.ok) {
        if (response.status === 401) {
          // unauthorized
        }
        throw new Error(`Unexpected response status code ${response.status}`);
      }
      const body = await response.json();
      dispatch(loadingData("finalizeAppSettings", false));
      return body;
    } catch (error) {
      dispatch(loadingData("finalizeAppSettings", false));
      console.log(error);
      return;
    }
  }
}

export function setApplicationState(payload) {
  return async (dispatch) => {
    dispatch(loadingData("setAppState", true));
    let response;
    try {
      const url = `${apiEndpoint}/state`;
      response = await fetch(url, {
        method: "PUT",
        body: JSON.stringify({
          runState: payload
        }),
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        }
      });
      if (!response.ok) {
        if (response.status === 401) {
          // unauthorized
        }
        throw new Error(`Unexpected response status code ${response.status}`);
      }
      const body = await response.json();
      console.log(body);
      dispatch(loadingData("setAppState", false));
    } catch (error) {
      dispatch(loadingData("setAppState", false));
      console.log(error);
      return;
    }
  }
}
