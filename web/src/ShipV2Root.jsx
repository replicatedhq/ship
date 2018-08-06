import React from "react";
import { Provider } from "react-redux";
import { hot } from "react-hot-loader";
import { getStore } from "./redux";
import RoutesContainer from "./containers/RoutesContainer";
import AppWrapper from "./containers/AppWrapper";
import "./scss/index.scss";

class ConfigRoot extends React.Component {
  render() {
    return (
      <Provider store={getStore()}>
        <AppWrapper>
          <RoutesContainer />
        </AppWrapper>
      </Provider>
    );
  }
}

export default hot(module)(ConfigRoot)