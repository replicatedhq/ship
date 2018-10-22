import React from "react";
import PropTypes from "prop-types";
import ConfigItemTitle from "../config_render/ConfigItemTitle";

export default class HelmReleaseNameInput extends React.Component {

  handleOnChange = (e) => {
    const { onChange } = this.props;
    onChange(e.target.value);
  }

  render() {
    const { value } = this.props;

    return (
      <div className={`field field-type-text container u-marginTop--15`}>
        <ConfigItemTitle
          title="Helm Name"
        />
        <p className="u-color--dustyGray u-fontSize--normal u-marginTop--normal u-marginBottom--20">This is the name that will be used to template your Helm chart.</p>
        <div className="field-input-wrapper u-marginTop--15">
          <input
            type={this.props.inputType}
            {...this.props.props}
            placeholder={this.props.default}
            value={value}
            onChange={this.handleOnChange}
            className={`${this.props.className || ""} Input`} />
        </div>
      </div>
    );
  }
}

HelmReleaseNameInput.propTypes = {
  value: PropTypes.string.isRequired,
  onChange: PropTypes.func.isRequired,
};
