import * as React from "react";
import autoBind from "react-autobind";
import { Link } from "react-router-dom";
import { HashLink } from "react-router-hash-link";

const stopPropagation = (e) => e.stopPropagation();
export default class SidebarItem extends React.Component {
  static defaultProps = {
    className: "",
    linkClassName: "",
    onClick: () => { return; },
  };

  constructor() {
    super();
    autoBind(this);
  }

  render() {
    const {
      isActive,
      label,
      className,
      linkClassName,
      linkTo,
      subItemLinkTo,
      onClick,
    } = this.props;

    return (
      <div
        className={`SidebarItem-wrapper u-position--relative ${isActive ? "is-active" : ""} ${className || ""}`}
        onClick={(e) => {
          onClick(e);
          stopPropagation(e);
        }}
      >
        <div className="SidebarItem">
          {linkTo ?
            <Link className={linkClassName} to={linkTo} tabIndex="-1">{label}</Link>
            : subItemLinkTo ?
              <HashLink className="SubItem-label" to={subItemLinkTo} scroll={el => el.scrollIntoView({ behavior: "smooth", block: "start" })} tabIndex="-1">{label}</HashLink>
              :
              <span className="SubItem-label">{label}</span>
          }
        </div>
      </div>
    );
  }
}
