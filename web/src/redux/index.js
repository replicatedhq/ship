import { createStore, combineReducers, applyMiddleware, compose } from "redux";
import { createTracker } from "redux-segment";
import thunk from "redux-thunk";

// Reducers
import DataReducers from "./data";
import UIReducers from "./ui";

const tracker = createTracker();

const appReducer = combineReducers({
  data: DataReducers,
  ui: UIReducers,
});

const rootReducer = (state, action) => {
  if (action.type === "PURGE_ALL") {
    state = undefined
  }
  return appReducer(state, action);
};

let store;
export function configStore() {
  const hasExtension = window.devToolsExtension;
  return new Promise((resolve, reject) => {
    try {
      store = createStore(
        rootReducer,
        compose(
          applyMiddleware(thunk, tracker),
          hasExtension ? window.devToolsExtension() : f => f,
        ),
      );
      resolve(store);
    } catch (error) {
      reject(error);
    }
  });
}

export function getStore() {
  return store;
}
