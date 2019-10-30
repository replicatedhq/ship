import React from "react";

export default class ConfigItemTitle extends React.Component {

  render() {
    const {
      title,
      recommended,
      required,
      hidden,
      when
    } = this.props;

    var isHidden = hidden || when === "false";

    return (
      <h4 className={`sub-header-color field-section-sub-header ${isHidden ? "hidden" : ""}`}>{title} {
        required ? 
          <span className="field-label required">Required</span> :
          recommended ? 
            <span className="field-label recommended">Recommended</span> :
            null}
      <span className="u-marginLeft--10 config-errblock" id={`${this.props.name}-errblock`}></span>
      </h4>
    );
  }
}
