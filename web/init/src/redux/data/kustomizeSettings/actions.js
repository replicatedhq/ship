import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
//import { Utilities } from "../../../utilities/utilities";

export const constants = {
  RECEIVE_METADATA: "RECEIVE_METADATA",
  SET_HELM_CHART_ERROR: "SET_HELM_CHART_ERROR"
};

export function receiveMetadata(payload) {
  return {
    type: constants.RECEIVE_METADATA,
    payload
  };
}

export function setHelmChartError(error) {
  return {
    type: constants.SET_HELM_CHART_ERROR,
    payload: error
  }
}

export function getMetadata(loaderType = "getMetadata") {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    let response;
    dispatch(loadingData(loaderType, true));
    try {
      const url = `${apiEndpoint}/metadata`;
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
      dispatch(receiveMetadata(body));
    } catch (error) {
      console.log(error);
      //   if (Utilities.isFailedToFetchErr(error.message)) {
      //     dispatch(receiveHelmChartMetadata({ currentStep: {}, phase: "loading"}));
      //   } else {
      //     dispatch(setHelmChartError(error.message));
      //   }
      return;
    }
  };
}

export function saveHelmChartValues(payload, loaderType = "saveHelmChartValues") {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    let response;
    dispatch(loadingData(loaderType, true));
    const url = `${apiEndpoint}/helm-values`;
    response = await fetch(url, {
      method: "POST",
      body: JSON.stringify(payload),
      headers: {
        "Accept": "application/json",
        "Content-Type": "application/json"
      }
    });
    if (!response.ok) {
      dispatch(loadingData(loaderType, false));
      if (response.status === 400) {
        return response.json();
      }
      throw new Error("Internal server error");
    }
    const body = await response.blob();
    dispatch(loadingData(loaderType, false));
    return body;
  };
}
