import React from "react";
import { Provider } from "react-redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import "./scss/index.scss";
import { configureStore } from "./redux";
import PropTypes from "prop-types";

export const Ship = ({ apiEndpoint }) => (
  <Provider store={configureStore(apiEndpoint)}>
    <AppWrapper>
      <RouteDecider />
    </AppWrapper>
  </Provider>
);

Ship.propTypes = {
  apiEndpoint: PropTypes.string.isRequired,
}
