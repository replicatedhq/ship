import * as React from "react";
import { withRouter } from "react-router-dom";
import Sidebar from "./containers/Sidebar";

class Layout extends React.Component {
  render() {
    const { location, configRouteId } = this.props;
    const hideSidebar = location.pathname.includes("/setup") || location.pathname === "/audit-log" || location.pathname === "/preflight-checks";
    return (
      <div className="flex flex1">
        <div className="u-minHeight--full u-minWidth--full flex-column flex1 u-position--relative">
          <div className="flex flex1 u-minHeight--full u-height--full">
            {hideSidebar ? null :
              <div className="flex-column flex1 Sidebar-wrapper u-overflow--hidden">
                <div className="flex-column flex1">
                  <Sidebar
                    configOnly={this.props.configOnly}
                    location={location}
                    configRouteId={configRouteId}
                  />
                </div>
              </div>
            }
            <div className="flex-column flex1 u-height--auto u-overflow--hidden LayoutContent-wrapper">
              {this.props.children}
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(Layout);
