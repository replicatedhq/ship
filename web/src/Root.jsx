import React from "react";
import { Provider } from "react-redux";
import { hot } from "react-hot-loader";
import { BrowserRouter, Route, Redirect } from "react-router-dom";
import { getStore } from "./redux";
import "./scss/index.scss";

import NavBar from "./containers/Navbar";
import Layout from "./Layout";
import ApplicationSettings from "./containers/ApplicationSettings";

class Root extends React.Component {
  render() {
    return (
      <Provider store={getStore()}>
        <BrowserRouter>
          <div className="u-minHeight--full u-minWidth--full flex-column flex1">
            <NavBar />
            <Layout>
              <Route exact path="/" component={() => <Redirect to="/application-settings" />} />
              <Route exact path="/application-settings" component={ApplicationSettings} />
            </Layout>
          </div>
        </BrowserRouter>
      </Provider>
    );
  }
}

export default hot(module)(Root)
