import React from "react";
import { Provider } from "react-redux";
import { hot } from "react-hot-loader";
import { getStore } from "./redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import "./scss/index.scss";

class ShipRoot extends React.Component {
  render() {
    return (
      <Provider store={getStore()}>
        <AppWrapper>
          <RouteDecider />
        </AppWrapper>
      </Provider>
    );
  }
}

export default hot(module)(ShipRoot)