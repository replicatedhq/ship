import * as React from "react";
import { Link } from "react-router-dom";
import Popover from "./Popover";
import PopoverItem from "./PopoverItem";

export default class NavItem extends React.Component {

  componentDidMount() {
    document.addEventListener("mousedown", this.handleClickOutside);
  }

  componentWillUnmount() {
    document.removeEventListener("mousedown", this.handleClickOutside);
  }

  setWrapperRef = (node) => {
    this.wrapperRef = node;
  }

  handleClickOutside = (event) => {
    if (this.props.isDropDownActive && this.wrapperRef && !this.wrapperRef.contains(event.target)) {
      if (this.props.dropdownContent) {
        this.props.onClick(event)
      }
    }
  }

  render() {
    const {
      href,
      isActive,
      icon,
      label,
      className,
      linkTo,
      onClick,
      options,
      dropdownContent,
      isDropDownActive,
    } = this.props;
    const wrapperClassName = [
      "NavItem u-position--relative flex flex-column flex-verticalCenter",
      className,
      isActive ? "is-active" : "",
    ].join(" ");

    function linkify(linkContent, linkClassName, options = {}) {
      return linkTo
        ? <Link className={linkClassName} to={linkTo} tabIndex="-1">{linkContent}</Link>
        : href ?
          <a href={href} target="_blank" tabIndex="-1" rel="noopener noreferrer">{linkContent}</a>
          : options.isButton ?
            <button className={`Button ${options.buttonClassName || ""}`} tabIndex="-1">{linkContent}</button>
            : <a tabIndex="-1">{linkContent}</a>;
    }

    const stopPropagation = (e) => e.stopPropagation();

    return (
      <div
        onClick={(e) => {
          onClick(e);
          stopPropagation(e);
        }}
        className={wrapperClassName}
        ref={this.setWrapperRef}
      >
        <div className={`HeaderContent-wrapper flex0 ${dropdownContent && isDropDownActive ? "active-dropdown" : ""}`}>
          {icon
            ? linkify(icon, "HeaderLink flex0")
            : null
          }
          {label
            ? linkify(label, "HeaderLink flex0", options)
            : null
          }
          {dropdownContent
            ? (
              <Popover
                minWidth={this.props.dropdownWidth || "170"}
                position={this.props.dropdownPosition || "bottom-left"}
                visible={isDropDownActive}
                onClick={stopPropagation}
                content={Array.isArray(dropdownContent)
                  ? (
                    <ul className="PopoverItem-wrapper">
                      {dropdownContent
                        .filter(x => x)
                        .map((contents, index) => (
                          <PopoverItem
                            key={`item-${index}`}
                            {...contents}
                          />
                        ))
                      }
                    </ul>
                  )
                  : dropdownContent
                }
              />
            )
            : null
          }
        </div>
      </div>
    );
  }
}
