import React from "react";
import { BrowserRouter, Route, Switch } from "react-router-dom";
import RouteDecider from "../shared/RouteDecider";
import NavBar from "../../containers/Navbar";
import ConfigOnly from "../../containers/ConfigOnly";

import Loader from "./Loader";
import StepNumbers from "./StepNumbers";
import DetermineComponentForRoute from "../../containers/DetermineComponentForRoute";

export default class RoutesContainer extends React.Component {
  
  componentDidMount() {
    this.props.getRoutes();
  }

  render() {
    const { dataLoading, routes } = this.props;
    return (
      <div className="flex-column flex1">
        <div className="flex-column flex1 u-overflow--hidden u-position--relative">
          {dataLoading.routesLoading ?
            <div className="flex1 flex-column justifyContent--center alignItems--center">
              <Loader size="60" />
            </div>
            :
            <BrowserRouter>
              <RouteDecider routes={routes}>
                <div className="u-minHeight--full u-minWidth--full flex-column flex1">
                  <NavBar hideLinks={true} />
                  <StepNumbers steps={this.props.routes} />
                  <div className="flex-1-auto flex-column u-overflow--auto">
                    <Switch>
                      <Route exact path="/application-settings" component={ConfigOnly} />
                      {routes && routes.map((route) => (
                        <Route
                          exact
                          key={route.id}
                          path={`/${route.id}`}
                          render={() => <DetermineComponentForRoute routes={routes} routeId={route.id} />}
                        />
                      ))}
                      <Route exact path="/" component={() => <div className="flex1 flex-column justifyContent--center alignItems--center"><Loader size="60" /></div> } />
                    </Switch>
                  </div>
                </div>
              </RouteDecider>
            </BrowserRouter>
          }
        </div>
      </div>
    );
  }
}