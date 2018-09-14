import React from "react";
// import ReactDOM from "react-dom";
import autoBind from "react-autobind";
import get from "lodash/get";

export default class ConfigRadio extends React.Component {

  constructor(props) {
    super(props);
    autoBind(this);
  }

  handleOnChange(e) {
    const { group } = this.props;
    if (this.props.handleChange && typeof this.props.handleChange === "function") {
      this.props.handleChange(group, e.target.value);
    }
  }

  render() {

    let val = get(this.props, "value");
    if (!val || val.length === 0) {
      val = this.props.default;
    }
    const checked = val === this.props.name;

    return (
      <div className="flex-auto flex u-marginRight--20">
        <input
          type="radio"
          name={this.props.group}
          id={this.props.name}
          value={this.props.name}
          checked={checked}
          disabled={this.props.readOnly}
          onChange={(e) => this.handleOnChange(e)}
          className={`${this.props.className || ""} flex-auto ${this.props.readOnly ? "readonly" : ""}`} />
        <label htmlFor={this.props.name} className={`u-marginLeft--small header-color field-section-sub-header u-userSelect--none ${this.props.readOnly ? "u-cursor--default" : "u-cursor--pointer"}`}>{this.props.title}</label>
      </div>
    );
  }
}
