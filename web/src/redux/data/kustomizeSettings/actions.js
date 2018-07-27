import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
//import { Utilities } from "../../../utilities/utilities";

const apiEndpoint = window.env.API_ENDPOINT;
export const constants = {
  RECEIVE_HELM_CHART_METADATA: "RECEIVE_HELM_CHART_METADATA",
  SET_HELM_CHART_ERROR: "SET_HELM_CHART_ERROR"
};

export function receiveHelmChartMetadata(payload) {
  return {
    type: constants.RECEIVE_HELM_CHART_METADATA,
    payload
  };
}

export function setHelmChartError(error) {
  return {
    type: constants.SET_HELM_CHART_ERROR,
    payload: error
  }
}

export function getHelmChartMetadata(loaderType = "getHelmChartMetadata") {
  return async (dispatch) => {
    let response;
    dispatch(loadingData(loaderType, true));
    try {
      const url = `${apiEndpoint}/helm-metadata`;
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
      dispatch(receiveHelmChartMetadata(body));
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
  return async (dispatch) => {
    let response;
    dispatch(loadingData(loaderType, true));
    try {
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
        return;
      }
      const body = await response.blob();
      dispatch(loadingData(loaderType, false));
      return body;
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
