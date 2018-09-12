import React from "react";
import { Provider } from "react-redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import "./scss/index.scss";
import { configureStore } from "./redux";
import PropTypes from "prop-types";

const ShipV2Root = ({ apiEndpoint }) => (
  <Provider store={configureStore(apiEndpoint)}>
    <AppWrapper>
      <RouteDecider />
    </AppWrapper>
  </Provider>
);

ShipV2Root.propTypes = {
  apiEndpoint: PropTypes.string.isRequired,
}

export default ShipV2Root;
