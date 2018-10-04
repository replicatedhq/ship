import React from "react";
import PropTypes from "prop-types";
import { Router, Route, Switch } from "react-router-dom";
import isEmpty from "lodash/isEmpty";
import NavBar from "../../containers/Navbar";

import Loader from "./Loader";
import StepNumbers from "./StepNumbers";
import DetermineComponentForRoute from "../../containers/DetermineComponentForRoute";
import StepDone from "./StepDone";

const isRootPath = (basePath) => {
  const formattedBasePath = basePath === "" ? "/" : basePath.replace(/\/$/, "");
  return window.location.pathname === formattedBasePath
}

const ShipRoutesWrapper = ({ routes, headerEnabled, basePath, onCompletion }) => (
  <div className="flex-column flex1">
    <div className="flex-column flex1 u-overflow--hidden u-position--relative">
      {!routes ?
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <Loader size="60" />
        </div>
        :
        <div className="u-minHeight--full u-minWidth--full flex-column flex1">
          {headerEnabled && <NavBar hideLinks={true} routes={routes} basePath={basePath} />}
          <StepNumbers basePath={basePath} steps={routes} />
          <div className="flex-1-auto flex-column u-overflow--auto">
            <Switch>
              {routes && routes.map((route) => (
                <Route
                  exact
                  key={route.id}
                  path={`${basePath}/${route.id}`}
                  render={() => <DetermineComponentForRoute
                    onCompletion={onCompletion}
                    basePath={basePath}
                    routes={routes}
                    routeId={route.id}
                  />}
                />
              ))}
              <Route exact path={`${basePath}/`} component={() => <div className="flex1 flex-column justifyContent--center alignItems--center"><Loader size="60" /></div> } />
              <Route exact path={`${basePath}/done`} component={() =>  <StepDone />} />
            </Switch>
          </div>
        </div>
      }
    </div>
  </div>
)

export default class RouteDecider extends React.Component {
  static propTypes = {
    isDone: PropTypes.bool.isRequired,
    routes: PropTypes.arrayOf(
      PropTypes.shape({
         id: PropTypes.string,
         description: PropTypes.string,
         phase: PropTypes.string,
      })
    ),
    basePath: PropTypes.string.isRequired,
    headerEnabled: PropTypes.bool.isRequired,
    history: PropTypes.object.isRequired,
    /** Callback function to be invoked at the finalization of the Ship Init flow */
    onCompletion: PropTypes.func,
  }

  componentDidUpdate(lastProps) {
    const {
      routes,
      getHelmChartMetadata,
      basePath
    } = this.props
    if (routes !== lastProps.routes && routes.length) {
      for (let i = 0; i < routes.length; i++) {
        if (routes[i].phase.includes("helm")) {
          getHelmChartMetadata();
          break;
        }
      }
      if (isRootPath(basePath)) {
        const defaultRoute = `${basePath}/${routes[0].id}`;
        this.props.history.push(defaultRoute);
      }
    }
  }

  componentDidMount() {
    if (isEmpty(this.props.routes)) {
      this.props.getRoutes();
    }
  }

  render() {
    const {
      routes,
      basePath,
      history,
      headerEnabled,
      onCompletion,
    } = this.props;
    const routeProps = {
      routes,
      basePath,
      headerEnabled,
      onCompletion,
    }
    return (
      <div className="u-minHeight--full u-minWidth--full flex-column flex1">
        <Router history={history}>
          <ShipRoutesWrapper
            {...routeProps}
          />
        </Router>
      </div>
    );
  }
}
