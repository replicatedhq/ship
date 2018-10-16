import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";
import { getContentForStep } from "../appRoutes/actions";

export const constants = {
  RECEIVE_FILE_CONTENT: "RECEIVE_FILE_CONTENT",
  RECEIVE_PATCH: "RECEIVE_PATCH",
  RECEIVE_MODIFIED: "RECEIVE_MODIFIED",
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
  return async (dispatch, getState) => {
    let response;
    dispatch(loadingData("fileContent", true));
    try {
      const { apiEndpoint } = getState();
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
      dispatch(receivePatch(body.overlay));
    } catch (error) {
      dispatch(loadingData("fileContent", false));
      console.log(error)
      return;
    }
  };
}

export function saveKustomizeOverlay(payload) {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
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
      dispatch(getFileContent(payload.path));
      dispatch(loadingData("saveKustomize", false));
    } catch (error) {
      dispatch(loadingData("saveKustomize", false));
      console.log(error)
      return;
    }
  };
}

export function deleteOverlay(path, isResource) {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    let response;
    let url = `${apiEndpoint}/kustomize/patch?path=${path}`;
    if(isResource) url = `${apiEndpoint}/kustomize/resource?path=${path}`;
    dispatch(loadingData("deleteOverlay", true));
    try {
      response = await fetch(url, {
        method: "DELETE",
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        }
      });
      if (!response.ok) {
        dispatch(loadingData("deleteOverlay", false));
        return;
      }
      await response.json();
      dispatch(loadingData("deleteOverlay", false));
      dispatch(receivePatch(""));
      dispatch(getFileContent(path));
      dispatch(getContentForStep("kustomize"));
    } catch (error) {
      dispatch(loadingData("deleteOverlay", false));
      console.log(error)
      return;
    }
  };
}

export function finalizeKustomizeOverlay() {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
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
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
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
      let { patch } = await response.json();
      if (!response.ok) {
        patch = payload.current;
      }
      dispatch(receivePatch(patch));
    } catch (error) {
      console.log(error)
      return;
    }
  };
}

export function applyPatch(payload) {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    try {
      const url = `${apiEndpoint}/kustomize/apply`;
      const response = await fetch(url, {
        method: "POST",
        headers: {
          "Accept": "application/json",
          "Content-Type": "application/json"
        },
        body: JSON.stringify(payload)
      });
      const { modified } = await response.json();
      dispatch(receiveModified(modified, payload.patch));
    } catch (error) {
      console.log(error)
      return;
    }
  };
}

export function receiveModified(modified, patch) {
  return {
    type: constants.RECEIVE_MODIFIED,
    payload: {
      modified,
      patch,
    }
  };
}
