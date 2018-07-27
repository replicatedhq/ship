export const  constants = {
  LOADING_DATA: "LOADING_DATA",
};

export function loadingData(key, isLoading) {
  return {
    type: constants.LOADING_DATA,
    payload: {
      key,
      isLoading,
    },
  };
}