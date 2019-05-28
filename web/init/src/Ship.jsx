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
    /** Determines whether steps are displayed below navbar or in navbar */
    stepsEnabled: PropTypes.bool,
    /** Parent history needed to sync Ship routing with parent<br>Note: Defaults to instantiate own internal BrowserRouter if omitted. */
    history: PropTypes.object,
    /** Callback function to be invoked at the finalization of the Ship Init flow */
    onCompletion: PropTypes.func,
    /** Callback function to be invoked when there's an unresolved error thrown followed componentDidCatch() method signature */
    onError: PropTypes.func
  }

  static defaultProps = {
    basePath: "",
    history: createBrowserHistory(),
    headerEnabled: false,
    stepsEnabled: false
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

  componentDidCatch(error, errorInfo) {
    const { onError } = this.props;
    if (onError) {
      onError(error, errorInfo);
    }
  }

  render() {
    const { history, headerEnabled, stepsEnabled, basePath, onCompletion } = this.props;
    const { store } = this.state;

    return (
      <div id="ship-init-component">
        <Provider store={store}>
          <AppWrapper>
            <RouteDecider
              headerEnabled={headerEnabled}
              stepsEnabled={stepsEnabled}
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
