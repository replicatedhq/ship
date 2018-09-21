import React from "react";
import ReactDOM from "react-dom";
import autoBind from "react-autobind";
import FileInput from "./FileInput";
import ConfigItemTitle from "./ConfigItemTitle";

export default class ConfigFileInput extends React.Component {

  constructor(props) {
    super(props);
    autoBind(this);
  }

  handleOnChange(value, data) {
    if (this.props.handleChange) {
      if (this.props.multiple) {
        this.props.handleChange(this.props.name, data, value);
      } else {
        this.props.handleChange(
          this.props.name,
          data ? data[0] : "",
          value ? value[0] : "",
        );
      }
    }
  }

  getFilenamesText() {
    if (this.props.multiple) {
      if (this.props.multi_value && this.props.multi_value.length) {
        return this.props.multi_value.join(", ");
      }
    } else if (this.props.value) {
      return this.props.value;
    }
    return this.props.default;
  }

  render() {
    return (
      <div className={`field field-type-file ${this.props.hidden ? "hidden" : ""}`}>
        {this.props.title !== "" ?
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
          />
          : null}
        <div className="input input-type-file clearfix">
          <div>
            <span>
              <FileInput
                ref={(file) => this.file = file}
                name={this.props.name}
                readOnly={this.props.readonly}
                disabled={this.props.readonly}
                multiple={this.props.multiple}
                onChange={this.handleOnChange} />
            </span>
            <small>{this.getFilenamesText()}</small>
          </div>
        </div>
      </div>
    );
  }
}
