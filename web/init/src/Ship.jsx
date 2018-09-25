import React from "react";
import { Provider } from "react-redux";
import RouteDecider from "./containers/RouteDecider";
import AppWrapper from "./containers/AppWrapper";
import { configureStore } from "./redux";
import PropTypes from "prop-types";

import "./scss/index.scss";

export class Ship extends React.Component {
  constructor() {
    super();
    this.state = {
      store: null
    }
  }
  static propTypes = {
    /** API endpoint for the Ship binary */
    apiEndpoint: PropTypes.string.isRequired,
    /**
     * Base path name for the internal Ship Init component router
     * */
    basePath: PropTypes.string,
    /**
     * Determines whether or not the Ship Init app will instantiate its own BrowserRouter
     * */
    headerEnabled: PropTypes.bool,
    /**
     * Parent history needed to sync ship routing with parent
     * */
    history: PropTypes.object
  }

  static defaultProps = {
    basePath: "",
    history: null,
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
    const { history, headerEnabled, basePath } = this.props;
    const { store } = this.state;

    return (
      <div id="ship-init-component">
        <Provider store={store}>
          <AppWrapper>
            <RouteDecider
              headerEnabled={headerEnabled}
              basePath={basePath}
              history={history}
            />
          </AppWrapper>
        </Provider>
      </div>
    )
  }
}
