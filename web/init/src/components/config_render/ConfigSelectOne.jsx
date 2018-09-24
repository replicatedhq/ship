import React from "react";
import map from "lodash/map";
import isEmpty from "lodash/isEmpty";

import ConfigItemTitle from "./ConfigItemTitle";
import ConfigRadio from "./ConfigRadio";

export default class ConfigSelectOne extends React.Component {

  handleOnChange = (itemName, val) => {
    if (this.props.handleOnChange && typeof this.props.handleOnChange === "function") {
      this.props.handleOnChange(itemName, val);
    }
  }

  render() {
    let options = [];
    map(this.props.items, (childItem, i) => {
      if (isEmpty(childItem)) return null;
      options.push(
        <ConfigRadio
          key={`${childItem.name}-${i}`}
          name={childItem.name}
          title={childItem.title}
          id={childItem.name}
          default={this.props.default}
          group={this.props.name}
          value={this.props.value}
          readOnly={this.props.readonly}
          handleChange={(itemName, val) => this.handleOnChange(itemName, val)}
        />
      )
    });

    return (
      <div className={`field field-type-select-one ${this.props.hidden ? "hidden" : "u-marginTop--15"}`}>
        {this.props.title !== "" ?
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
          />
          : null}
        {this.props.help_text !== "" ? <p className="field-section-help-text u-marginTop--small u-lineHeight--normal">{this.props.help_text}</p> : null}
        <div className="field-input-wrapper u-marginTop--15 flex">
          {options}
        </div>
      </div>
    );
  }
}
