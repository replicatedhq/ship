import React, { Fragment } from "react";
import assign from "object-assign";
import { Link, withRouter } from "react-router-dom";
import StepNumbers from "./StepNumbers";
import upperFirst from "lodash/upperFirst";
import NavItem from "./NavItem";
// This is hardcoded for now as we're bundling it from `@replicatedhq/ship-init`
// and then re-bundling the svg as a part of the `@replicatedhq/ship-app` bundle
// for it to be served via the ship binary. As part of the bundling process the name
// is mutated to a data-uri twice.
// Do not remove the import.
import "../../assets/images/ship-logo.png";
const shipLogo = "static/media/b3d517c0409239a363a3c18ce9a0eda2.b3d517c0.png";
export class NavBar extends React.Component {

  constructor() {
    super();
    this.state = {
      navDetails: {
        name: "",
        icon: "",
      },
      imageLoaded: false,
    };
  }

  isActive = (pathname = "") => {
    return (item = {}) => {
      if (!item.linkTo) return false;
      return pathname.indexOf(`${item.linkTo}`) > -1;
    };
  }

  handleRouteChange = (route, dropdownKey) => {
    if (this.state[`${dropdownKey}Active`]) {
      this.setState({
        [`${dropdownKey}Active`]: false
      });
    }
    const { basePath } = this.props;
    this.props.history.push(`${basePath}/${route}`);
  }

  handleLogOut = (e) => {
    e.preventDefault();
  }

  getNavItems = () => {
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

  combineItems = (methods) => {
    return methods.reduce((accum, method) => (
      accum.concat(method(this.props))
    ), []);
  }

  onClick = (item) => {
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

  preloadNavIconImage = (iconUrl) => new Promise(
    (resolve, reject) => {
      var image = new Image();
      image.onload = resolve;
      image.onerror = reject;
      image.src = iconUrl;
    }
  )

  componentDidUpdate() {
    const { shipAppMetadata } = this.props;
    const { imageLoaded } = this.state;

    if (!imageLoaded && shipAppMetadata.loaded) {
      this.preloadNavIconImage(shipAppMetadata.icon)
        .then(() => {
          this.setState({
            navDetails: {
              name: shipAppMetadata.name,
              icon: shipAppMetadata.icon,
            },
            imageLoaded: true,
          })
        })
        .catch(() => this.setState({
          navDetails: {
            name: shipAppMetadata.name,
            icon: shipLogo,
          },
          imageLoaded: true,
        }));
    }
  }

  render() {
    const { className, routes, basePath } = this.props;
    const { navDetails, imageLoaded } = this.state;
    const isPathActive = this.isActive(
      typeof window === "object"
        ? window.location.pathname
        : "",
    );

    const itemsArr = [this.getNavItems.bind(this)];
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

    const [ firstRoute = {} ] = routes;
    const { id: firstRouteId } = firstRoute;

    const headerLogo = (
      <div className="HeaderLogo-wrapper flex-column flex1 flex-verticalCenter u-position--relative">
        <div className="HeaderLogo">
          <Link to={`/${firstRouteId}`} tabIndex="-1">
            <img src={navDetails.icon} className="logo" />
          </Link>
        </div>
      </div>
    );

    const headerName = navDetails && navDetails.icon ? null : (
      <div className="flex-column flex-auto HeaderName-wrapper">
        {navDetails.name && navDetails.name.length ?
          <div className="flex-column flex1 flex-verticalCenter u-position--relative">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tundora u-lineHeight--default u-marginRight--30">{upperFirst(navDetails.name)}</p>
          </div>
          : <div className="flex-column flex1 flex-verticalCenter u-position--relative">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tundora u-lineHeight--default u-marginRight--30">Replicated Ship</p>
          </div>
        }
      </div>
    );

    return (
      <div className={`NavBarWrapper flex flex-auto ${className || ""}`}>
        <div className="container flex flex1">
          <div className="flex1 flex justifyContent--center alignItems--center">
            <div className="flex1 flex">
              <div className="flex flex-auto metadata-wrapper">
                {
                  imageLoaded ?
                    (
                      <Fragment>
                        {headerLogo}
                        {headerName}
                      </Fragment>
                    ) :
                    null
                }
                {this.props.hideLinks ? null :
                  <div className="flex flex-auto alignItems--center left-items">
                    {leftItems.map(renderItem)}
                  </div>
                }
              </div>
              {this.props.hideSteps ? null :
                <div className="flex flex1">
                  <StepNumbers basePath={basePath} steps={routes} inNav={true} />
                </div>
              }
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
