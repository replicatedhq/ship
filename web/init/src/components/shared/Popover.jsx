import * as React from "react";

export default class Popover extends React.Component {
  render() {
    const {
      className,
      visible,
      text,
      content,
      position,
      onClick,
      minWidth,
    } = this.props;

    const wrapperClass = `Popover-wrapper popover-${position || ""} ${className || ""} ${visible ? "is-active" : ""}`;

    return (
      <div className={wrapperClass} style={{ minWidth: `${minWidth}px` }} onClick={onClick}>
        <div className="Popover-content">
          {content || text}
        </div>
      </div>
    );
  }
}
