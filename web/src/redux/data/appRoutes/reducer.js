import { constants } from "./actions";

const routesDataState = {
  routes: []
};

export function routesData(state = routesDataState, action) {
  switch (action.type) {
  case constants.RECEIVE_ROUTES:
    return Object.assign({}, state, {
      routes: action.payload
    });
  default:
    return state;
  }
}
