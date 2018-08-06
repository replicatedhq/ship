import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
import { getCurrentStep } from "../determineSteps/actions";

const apiEndpoint = window.env.API_ENDPOINT;
export const constants = {
  RECEIVE_FILE_CONTENT: "RECEIVE_FILE_CONTENT",
  RECEIVE_PATCH: "RECEIVE_PATCH",
};

export function receiveFileContent(content, path) {
  return {
    type: constants.RECEIVE_FILE_CONTENT,
    payload: {
      content,
      path,
    }
  };
}

export function getFileContent(payload) {
  return async (dispatch) => {
    let response;
    dispatch(loadingData("fileContent", true));
    try {
      const url = `${apiEndpoint}/kustomize/file`;
      response = await fetch(url, {
        method: "POST",
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        },
        body: JSON.stringify({ path: payload })
      });
      if (!response.ok) {
        dispatch(loadingData("fileContent", false));
        return;
      }
      const body = await response.json();
      dispatch(loadingData("fileContent", false));
      dispatch(receiveFileContent(body, payload));
    } catch (error) {
      dispatch(loadingData("fileContent", false));
      console.log(error)
      return;
    }
  };
}

export function saveKustomizeOverlay(payload) {
  return async (dispatch) => {
    let response;
    dispatch(loadingData("saveKustomize", true));
    try {
      const url = `${apiEndpoint}/kustomize/save`;
      response = await fetch(url, {
        method: "POST",
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        },
        body: JSON.stringify(payload)
      });
      if (!response.ok) {
        dispatch(loadingData("saveKustomize", false));
        return;
      }
      await response.json();
      dispatch(loadingData("saveKustomize", false));
      dispatch(getCurrentStep());
    } catch (error) {
      dispatch(loadingData("saveKustomize", false));
      console.log(error)
      return;
    }
  };
}

export function finalizeKustomizeOverlay() {
  return async (dispatch) => {
    let response;
    dispatch(loadingData("finalizeKustomize", true));
    try {
      const url = `${apiEndpoint}/kustomize/finalize`;
      response = await fetch(url, {
        method: "POST",
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        }
      });
      if (!response.ok) {
        dispatch(loadingData("finalizeKustomize", false));
        return;
      }
      await response.json();
      dispatch(loadingData("finalizeKustomize", false));
    } catch (error) {
      dispatch(loadingData("finalizeKustomize", false));
      console.log(error)
      return;
    }
  };
}

export function receivePatch(patch) {
  return {
    type: constants.RECEIVE_PATCH,
    payload: {
      patch,
    }
  };
}

export function generatePatch(payload) {
  return async (dispatch) => {
    try {
      const url = `${apiEndpoint}/kustomize/patch`;
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        },
        body: JSON.stringify(payload)
      });
      const { patch } = await response.json();
      dispatch(receivePatch(patch));
    } catch (error) {
      console.log(error)
      return;
    }
  };
}
