import React from "react";
import ConfigItemTitle from "./ConfigItemTitle";

export default class ConfigInput extends React.Component {

  handleOnChange = (e) => {
    const { handleOnChange, name } = this.props;
    if (handleOnChange && typeof handleOnChange === "function") {
      handleOnChange(name, e.target.value);
    }
  }

  render() {
    return (
      <div className={`field field-type-text ${this.props.hidden ? "hidden" : "u-marginTop--15"}`}>
        {this.props.title !== "" ?
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
          />
          : null}
        {this.props.help_text !== "" ? <p className="field-section-help-text u-marginTop--small u-lineHeight--normal">{this.props.help_text}</p> : null}
        <div className="field-input-wrapper u-marginTop--15">
          <input
            type={this.props.inputType}
            {...this.props.props}
            placeholder={this.props.default}
            value={this.props.value}
            readOnly={this.props.readonly}
            disabled={this.props.readonly}
            onChange={(e) => this.handleOnChange(e)}
            className={`${this.props.className || ""} Input ${this.props.readonly ? "readonly" : ""}`} />
        </div>
      </div>
    );
  }
}
