import React from "react";
import { Provider } from "react-redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import { configureStore } from "./redux";
import PropTypes from "prop-types";

import "./scss/index.scss";
const bodyClass = "ship-init";

export class Ship extends React.Component {
  static propTypes = {
    /** API endpoint for the Ship binary */
    apiEndpoint: PropTypes.string.isRequired,
    /**
     * Base path name for the internal Ship Init component router
     * */
    basePath: PropTypes.string.isRequired,
  }

  componentDidMount() {
    document.body.classList.add(bodyClass);
  }
  componentWillUnmount() {
    document.body.classList.remove(bodyClass);
  }

  render() {
    const { apiEndpoint, basePath } = this.props;

    return (
      <div id="ship-init-component">
        <Provider store={configureStore(apiEndpoint)}>
          <AppWrapper>
            <RouteDecider basePath={basePath} />
          </AppWrapper>
        </Provider>
      </div>
    )
  }
}
