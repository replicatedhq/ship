import React from "react";
import { Provider } from "react-redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import { configureStore } from "./redux";
import PropTypes from "prop-types";

import "./scss/index.scss";
const bodyClass = "ship-init";

export class Ship extends React.Component {
  componentDidMount() {
    document.body.classList.add(bodyClass);
  }
  componentWillUnmount() {
    document.body.classList.remove(bodyClass);
  }

  render() {
    const { apiEndpoint } = this.props;

    return (
      <div id="ship-init-component">
        <Provider store={configureStore(apiEndpoint)}>
          <AppWrapper>
            <RouteDecider />
          </AppWrapper>
        </Provider>
      </div>
    )
  }
}

Ship.propTypes = {
  apiEndpoint: PropTypes.string.isRequired,
}
