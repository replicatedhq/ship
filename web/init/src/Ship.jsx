import React from "react";
import { Provider } from "react-redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import { configureStore } from "./redux";
import PropTypes from "prop-types";

import "./scss/index.scss";

export const Ship = ({ apiEndpoint }) => (
  <div id="ship-init-component">
    <Provider store={configureStore(apiEndpoint)}>
      <AppWrapper>
        <RouteDecider />
      </AppWrapper>
    </Provider>
  </div>
);

Ship.propTypes = {
  apiEndpoint: PropTypes.string.isRequired,
}
