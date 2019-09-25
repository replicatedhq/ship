import React from "react";
import ConfigRender from "./components/config_render/ConfigRender";
import PropTypes from "prop-types";
import map from "lodash/map";
import sortBy from "lodash/sortBy";
import keyBy from "lodash/keyBy";

import "./scss/index.scss";

export class ShipConfigRenderer extends React.Component {
  static propTypes = {
    groups: PropTypes.array.isRequired, // Config groups items to render
    handleChange: PropTypes.func,
    getData: PropTypes.func,
  }

  static defaultProps = {
    groups: []
  }

  constructor(props) {
    super(props);
  }

  render() {
    const { groups } = this.props;
    const orderedFields = sortBy(groups, "position");
    const _groups = keyBy(orderedFields, "name");
    const groupsList = map(groups, "name");

    return (
      <div id="ship-init-component">
        <ConfigRender
          fieldsList={groupsList}
          fields={_groups}
          handleChange={this.props.handleChange}
        />
      </div>
    )
  }
}
