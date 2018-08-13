import { constants } from "./actions";

const routesDataState = {
  routes: [],
  isKustomizeFlow: false
};

export function routesData(state = routesDataState, action) {
  switch (action.type) {
  case constants.RECEIVE_ROUTES:
    let isKustomize = false;
    for (let i = 0; i < action.payload.length; i++) {
      if (action.payload[i].phase.includes("helm")) {
        isKustomize = true;
        break;
      }
    }
    return Object.assign({}, state, {
      routes: action.payload,
      isKustomizeFlow: isKustomize
    });
  default:
    return state;
  }
}
