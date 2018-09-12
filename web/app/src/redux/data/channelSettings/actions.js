import "isomorphic-fetch";
import { loadingData } from "../../ui/main/actions";

export const constants = {
  RECEIVE_CHANNEL_DETAILS: "RECEIVE_CHANNEL_DETAILS"
};

export function receiveChannelSettings(message) {
  return {
    type: constants.RECEIVE_CHANNEL_DETAILS,
    payload: message
  }
}

export function getChannel() {
  return async (dispatch, getState) => {
    const { apiEndpoint } = getState();
    let response;
    dispatch(loadingData("getChannel", true));
    try {
      const url = `${apiEndpoint}/channel`;
      response = await fetch(url, {
        method: "GET",
        headers: {
          "Accept": "application/json",
        },
      });
      if (!response.ok) {
        dispatch(loadingData("getChannel", false));
        return;
      }
      const body = await response.json();
      dispatch(loadingData("getChannel", false));
      dispatch(receiveChannelSettings(body));
    } catch (error) {
      dispatch(loadingData("getChannel", false));
      return;
    }
  };
}
