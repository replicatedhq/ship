import React from "react";
import { BrowserRouter, Route, Switch } from "react-router-dom";
import isEmpty from "lodash/isEmpty";
import NavBar from "../../containers/Navbar";

import Loader from "./Loader";
import StepNumbers from "./StepNumbers";
import DetermineComponentForRoute from "../../containers/DetermineComponentForRoute";
import StepDone from "./StepDone";

export default class RouteDecider extends React.Component {

  componentDidUpdate(lastProps) {
    if (this.props.routes !== lastProps.routes && this.props.routes.length) {
      for (let i = 0; i < this.props.routes.length; i++) {
        if (this.props.routes[i].phase.includes("helm")) {
          this.props.getHelmChartMetadata();
          break;
        }
      }
      if (window.location.pathname === "/") {
        window.location.replace(`/${this.props.routes[0].id}`);
      }
    }
  }

  componentDidMount() {
    if (isEmpty(this.props.routes)) {
      this.props.getRoutes();
    }
  }

  render() {
    const { routes, isDone } = this.props;
    const isOnRoot = window.location.pathname === "/";

    return (
      <div className="u-minHeight--full u-minWidth--full flex-column flex1">
        <BrowserRouter>
          <div className="flex-column flex1">
            <div className="flex-column flex1 u-overflow--hidden u-position--relative">
              {!routes ?
                <div className="flex1 flex-column justifyContent--center alignItems--center">
                  <Loader size="60" />
                </div>
                :
                <div className="u-minHeight--full u-minWidth--full flex-column flex1">
                  {isOnRoot ? null : <NavBar hideLinks={true} routes={routes} />}
                  {isOnRoot || isDone ? null : <StepNumbers steps={routes} />}
                  <div className="flex-1-auto flex-column u-overflow--auto">
                    <Switch>
                      {routes && routes.map((route) => (
                        <Route
                          exact
                          key={route.id}
                          path={`/${route.id}`}
                          render={() => <DetermineComponentForRoute routes={routes} routeId={route.id} />}
                        />
                      ))}
                      <Route exact path="/" component={() => <div className="flex1 flex-column justifyContent--center alignItems--center"><Loader size="60" /></div> } />
                      <Route exact path="/done" component={() =>  <StepDone />} />
                    </Switch>
                  </div>
                </div>
              }
            </div>
          </div>
        </BrowserRouter>
      </div>
    );
  }
}
