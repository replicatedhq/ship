import React, { Component, Fragment } from "react";
import PropTypes from "prop-types";
import classNames from "classnames";

export default class TabView extends Component {
  constructor(props) {
    super(props);
    console.log('constructor');
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
    ])
  }

  setTab = name => {

    this.setState({
      currentTab: name
    });
  }

  render() {
    const {
      className,
      children
    } = this.props;
    const { currentTab } = this.state;
    const childToRender = React.Children.toArray(children).find(child => child.props.name === currentTab);
    return (
      <div className={this.props.className}>
        <div className="tabview-tabwrapper">
          {React.Children.map(children, (child, idx ) => {
            const { displayText, name } = child.props;
            return (
              <Fragment>
                <span key={name} className={classNames("tabview-tabname", {
                  selected: name === currentTab
                })} onClick={() => { this.setTab(name); }}>
                  {displayText}
                </span>
                {idx + 1 !== children.length && "|"}
              </Fragment>
            );
          })}
        </div>
        {childToRender}
      </div>
    );
  }
}
