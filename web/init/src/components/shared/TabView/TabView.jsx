import React, { Component, Fragment } from "react";
import PropTypes from "prop-types";
import classNames from "classnames";

import "../../../scss/components/shared/TabView.scss";

export default class TabView extends Component {
  constructor(props) {
    super(props);
    const { children, initialTab } = props;
    const tabToDisplay = initialTab || children[0].props.name

    this.state = {
      currentTab: tabToDisplay
    };
  }

  static propTypes = {
    children: PropTypes.oneOfType([
      PropTypes.element,
      PropTypes.arrayOf(PropTypes.element)
    ]),
    separator: PropTypes.string
  }

  static defaultProps = {
    separator: "|"
  }

  setTab = name => {

    this.setState({
      currentTab: name
    });
  }

  render() {
    const {
      className,
      children,
      separator
    } = this.props;
    const { currentTab } = this.state;
    const childToRender = React.Children.toArray(children).find(child => child.props.name === currentTab);
    return (
      <div className={classNames("tabview", className)}>
        <div className="tabview-tabwrapper">
          {React.Children.map(children, (child, idx ) => {
            const { displayText, name } = child.props;
            return (
              <Fragment>
                <span key={name} className={classNames("tabview-tabname u-cursor--pointer", {
                  selected: name === currentTab
                })} onClick={() => { this.setTab(name); }}>
                  {displayText}
                </span>
                {idx + 1 !== children.length && separator}
              </Fragment>
            );
          })}
        </div>
        {childToRender}
      </div>
    );
  }
}
