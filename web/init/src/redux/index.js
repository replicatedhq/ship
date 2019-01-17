import { createStore, combineReducers, applyMiddleware, compose } from "redux";
import { createTracker } from "redux-segment";
import thunk from "redux-thunk";

// Reducers
import DataReducers from "./data";
import UIReducers from "./ui";

const tracker = createTracker();

let store;

export function configureStore(apiEndpoint) {
  const appReducer = combineReducers({
    data: DataReducers,
    ui: UIReducers,
    apiEndpoint: () => apiEndpoint
  });

  const rootReducer = (state, action) => {
    if (action.type === "PURGE_ALL") {
      state = undefined
    }
    return appReducer(state, action);
  };

  const hasExtension = window.devToolsExtension;
  store = createStore(
    rootReducer,
    compose(
      applyMiddleware(thunk, tracker),
      hasExtension ? window.devToolsExtension() : f => f,
    ),
  )

  return store;
}

export function getStore() {
  return store;
}
