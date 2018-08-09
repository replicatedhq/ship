import * as React from "react";
import { Link } from "react-router-dom";

export default class PopoverItem extends React.Component {
  render() {
    const {
      className,
      label,
      href,
      linkTo,
      subtext,
      icon,
      onClick,
    } = this.props;

    return (
      <li className={`PopoverItem ${className || ""}`}>
        {onClick ?
          <div className="u-noSelect flex-column flex1" onClick={onClick}>
            <div className="flex1 flex PopoverLabel">
              {icon ? <div className="PopoverIcon flex-auto">{icon}</div> : null}
              <div className="flex1 flex-column flex-verticalCenter PopoverTitle">{label}</div>
            </div>
            {subtext ?
              <div className="flex1 PopoverSub">{subtext}</div>
              : null}
          </div>
          : linkTo ?
            <Link className="PopoverLabel u-noSelect" to={linkTo}>
              {label}
            </Link>
            :
            <a className="PopoverLabel u-noSelect" href={href} target="_blank" rel="noopener noreferrer">
              {label}
            </a>
        }
      </li>
    );
  }
}
