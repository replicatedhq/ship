import React from "react";
import { Provider } from "react-redux";
import { hot } from "react-hot-loader";
import { BrowserRouter, Route, Switch } from "react-router-dom";
import { getStore } from "./redux";
import NavBar from "./containers/Navbar";
import AppWrapper from "./containers/AppWrapper";
import "./scss/index.scss";

import DetermineStep from "./containers/DetermineStep";
import ConfigOnly from "./containers/ConfigOnly";
import KustomizeOverlay from "./containers/KustomizeOverlay";

class ConfigRoot extends React.Component {
  render() {
    return (
      <Provider store={getStore()}>
        <AppWrapper>
          <BrowserRouter>
            <div className="u-minHeight--full u-minWidth--full flex-column flex1">
              <NavBar hideLinks={true} />
              <Switch>
                <Route exact path="/" component={DetermineStep} />
                <Route exact path="/application-settings" component={ConfigOnly} />
                <Route exact path="/kustomize" component={KustomizeOverlay} />
              </Switch>
            </div>
          </BrowserRouter>
        </AppWrapper>
      </Provider>
    );
  }
}

export default hot(module)(ConfigRoot)
