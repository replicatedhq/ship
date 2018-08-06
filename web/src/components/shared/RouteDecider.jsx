import React from "react";
import { withRouter } from "react-router-dom"
import isEmpty from "lodash/isEmpty";

class RouteDecider extends React.Component {

  componentDidUpdate(lastProps) {
    if (this.props.routes !== lastProps.routes && !isEmpty(this.props.routes)) {
      // this doesn't work...it always goes back to this route no matter what you type in.
      // this.props.history.push(`/${this.props.routes[0].id}`);
    }
  }

  render() {
    return (
      <div className="u-minHeight--full u-minWidth--full flex-column flex1">
        {this.props.children}
      </div>
    );
  }
}

export default withRouter(RouteDecider);