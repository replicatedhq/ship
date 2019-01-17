import React from "react";
import PropTypes from "prop-types";
import ConfigItemTitle from "../config_render/ConfigItemTitle";

export default class HelmAdvancedInput extends React.Component {

  handleOnChange = (e) => {
    const { onChange } = this.props;
    onChange(e.target.value);
  }

  render() {
    const {
      value,
      title,
      subTitle,
      placeholder,
    } = this.props;

    return (
      <div className={`field field-type-text u-marginTop--15`}>
        <ConfigItemTitle
          title={title}
        />
        <p className="u-color--dustyGray u-fontSize--normal u-marginTop--normal u-marginBottom--20">{subTitle}</p>
        <div className="field-input-wrapper u-marginTop--15">
          <input
            type={this.props.inputType}
            {...this.props.props}
            placeholder={placeholder}
            value={value}
            onChange={this.handleOnChange}
            className={`${this.props.className || ""} Input`} />
        </div>
      </div>
    );
  }
}

HelmAdvancedInput.propTypes = {
  value: PropTypes.string.isRequired,
  onChange: PropTypes.func.isRequired,
  title: PropTypes.string.isRequired,
  subTitle: PropTypes.string.isRequired,
  placeholder: PropTypes.string,
};
