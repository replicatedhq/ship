import React from "react";
import { Provider } from "react-redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import { configureStore } from "./redux";
import PropTypes from "prop-types";
import createBrowserHistory from 'history/createBrowserHistory'

import "./scss/index.scss";

export class Ship extends React.Component {
  static propTypes = {
    /** API endpoint for the Ship binary */
    apiEndpoint: PropTypes.string.isRequired,
    /** Base path name for the internal Ship Init component router<br>Note: If basePath is omitted, it will default the base route to "/" */
    basePath: PropTypes.string,
    /** Determines whether default header is displayed */
    headerEnabled: PropTypes.bool,
    /** Parent history needed to sync Ship routing with parent<br>Note: Defaults to instantiate own internal BrowserRouter if omitted. */
    history: PropTypes.object,
    /** Callback function to be invoked at the finalization of the Ship Init flow */
    onCompletion: PropTypes.func,
  }

  static defaultProps = {
    basePath: "",
    history: createBrowserHistory(),
    headerEnabled: false
  }

  constructor(props) {
    super(props);

    this.state = {
      store: configureStore(props.apiEndpoint),
    };
  }

  componentDidUpdate(prevProps) {
    const { apiEndpoint: previousApiEndpoint } = prevProps;
    const { apiEndpoint: currentApiEndpoint } = this.props;

    if (previousApiEndpoint !== currentApiEndpoint) {
      this.setState({
        store: configureStore(apiEndpoint)
      });
    }
  }

  render() {
    const { history, headerEnabled, basePath, onCompletion } = this.props;
    const { store } = this.state;

    return (
      <div id="ship-init-component">
        <Provider store={store}>
          <AppWrapper>
            <RouteDecider
              headerEnabled={headerEnabled}
              basePath={basePath}
              history={history}
              onCompletion={onCompletion}
            />
          </AppWrapper>
        </Provider>
      </div>
    )
  }
}
