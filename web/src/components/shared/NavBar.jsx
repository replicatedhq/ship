import * as React from "react";
import assign from "object-assign";
import autoBind from "react-autobind";
import { Link, withRouter } from "react-router-dom";
import upperFirst from "lodash/upperFirst";

import NavItem from "./NavItem";
import "../../scss/components/shared/NavBar.scss";
import shipLogo from "../../assets/images/ship-logo.png";

class NavBar extends React.Component {

  constructor() {
    super();
    this.state = {
      navDetails: {
        name: "",
        icon: ""
      }
    };
    autoBind(this);
  }

  isActive(pathname = "") {
    return (item = {}) => {
      if (!item.linkTo) return false;
      return pathname.indexOf(`${item.linkTo}`) > -1;
    };
  }

  handleRouteChange(route, dropdownKey) {
    if (this.state[`${dropdownKey}Active`]) {
      this.setState({
        [`${dropdownKey}Active`]: false
      });
    }
    this.props.history.push(route);
  }

  handleLogOut(e) {
    e.preventDefault();
    console.log("log out here")
  }

  getNavItems() {
    const token = true;

    return[ !token ? null :
      {
        id: 0,
        label: "Dashboard",
        linkTo: "/dashboard",
        isActive: this.props.location.pathname === "/dashboard",
        position: "left",
      },
    {
      id: 1,
      label: "Audit log",
      linkTo: "/audit-log",
      isActive: this.props.location.pathname === "/audit-log",
      position: "left",
    },
    !token ? null : {
      id: 2,
      label: "Logout",
      onClick: (e) => { this.handleLogOut(e) },
      position: "right"
    }
    ];
  }

  combineItems(methods) {
    return methods.reduce((accum, method) => (
      accum.concat(method(this.props))
    ), []);
  }

  onClick(item) {
    return (e, ...rest) => {
      const activeKey = `${item.dropdownLabel || item.id || ""}Active`;
      if (item.href) return;
      if (typeof item.onClick === "function") {
        item.onClick(e, ...rest);
        return;
      }
      this.setState({
        [activeKey]: !this.state[activeKey]
      });
    };
  }

  componentDidUpdate(lastProps) {
    if(this.props.phase !== lastProps.phase && this.props.phase !== "loading") {
      if(this.props.phase.includes("helm")) {
        this.setState({ 
          navDetails: {
            name: this.props.helmChartMetadata.name,
            icon: this.props.helmChartMetadata.icon,
          } 
        });
      } else {
        this.setState({ 
          navDetails: {
            name: this.props.channelDetails.channelName,
            icon: this.props.channelDetails.icon,
          } 
        });
      }
    }
  }

  render() {
    const { className } = this.props;
    const { navDetails } = this.state;
    const isPathActive = this.isActive(
      typeof window === "object"
        ? window.location.pathname
        : "",
    );
    
    const itemsArr = [];
    itemsArr.push(this.getNavItems);
    // build items
    const headerItems = this.combineItems(itemsArr)
      .filter(item => item)
      .map(item => (assign(item, {
        isActive: isPathActive(item),
      })));
    const renderItem = item => {
      return (
        <NavItem
          key={item.id}
          {...item}
          onClick={this.onClick(item)}
          isDropDownActive={this.state[`${item.dropdownLabel || item.id || ""}Active`]}
        />
      );
    };

    const rightItems = headerItems.filter(item => item.position === "right");
    const leftItems = headerItems.filter(item => item.position === "left");

    return (
      <div className={`NavBarWrapper flex flex-auto ${className || ""}`}>
        <div className="container flex flex1">
          <div className="flex1 justifyContent--flexStart alignItems--center">
            <div className="flex1 flex">
              <div className="flex flex-auto">
                <div className="HeaderLogo-wrapper flex-column flex1 flex-verticalCenter u-position--relative">
                  <div className="HeaderLogo">
                    <Link to="/" tabIndex="-1">
                      <img src={navDetails.icon ? navDetails.icon : shipLogo} className="logo" />
                    </Link>
                  </div>
                </div>
                <div className="flex-column flex-auto HeaderName-wrapper">
                  {navDetails.name && navDetails.name.length ?
                    <div className="flex-column flex1 flex-verticalCenter u-position--relative">
                      <p className="u-fontSize--larger u-fontWeight--bold u-color--tundora u-lineHeight--default u-marginRight--50">{upperFirst(navDetails.name)}</p>
                    </div>
                    : null}
                </div>
                {this.props.hideLinks ? null :
                  <div className="flex flex-auto alignItems--center left-items">
                    {leftItems.map(renderItem)}
                  </div>
                }
              </div>
              {this.props.hideLinks ? null :
                <div className="flex flex1 justifyContent--flexEnd right-items">
                  <div className="flex flex-auto alignItems--center">
                    {rightItems.map(renderItem)}
                  </div>
                </div>
              }
            </div>
          </div>
        </div>
      </div>
    );
  }
}


export default withRouter(NavBar);
