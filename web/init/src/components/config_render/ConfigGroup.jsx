import React from "react";
import ReactDOM from "react-dom"
import Markdown from "react-remarkable";
import each from "lodash/each";
import some from "lodash/some";
import isEmpty from "lodash/isEmpty";
import { ConfigService } from "../../services/ConfigService";

import ConfigInput from "./ConfigInput";
import ConfixTextarea from "./ConfigTextarea";
import ConfigSelectOne from "./ConfigSelectOne";
import ConfigItemTitle from "./ConfigItemTitle";
import ConfigCheckbox from "./ConfigCheckbox";
import ConfigFileInput from "./ConfigFileInput";

export default class ConfigGroup extends React.Component {

  handleItemChange = (itemName, value, data) => {
    if (this.props.handleChange) {
      this.props.handleChange(itemName, value, data);
    }
  }

  renderConfigItems = (items) => {
    if (!items) return null;
    return items.map((item, i) => {
      switch (item.type) {
      case "text":
        return (
          <ConfigInput
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            inputType="text"
            hidden={item.hidden}
            {...item}
          />
        );
      case "textarea":
        return (
          <ConfixTextarea
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            {...item}
          />
        );
      case "bool":
        return (
          <ConfigCheckbox
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            {...item}
          />
        );
      case "label":
        return (
          <div key={`${i}-${item.name}`} className="field field-type-label u-marginTop--15">
            <ConfigItemTitle
              title={item.title}
              recommended={item.recommended}
              required={item.required}
              hidden={item.hidden}
              name={item.name}
            />
          </div>
        );
      case "file":
        return (
          <div key={`${i}-${item.name}`}>
            <ConfigFileInput
              {...item}
              title={item.title}
              recommended={item.recommended}
              required={item.required}
              handleChange={this.handleItemChange}
              hidden={item.hidden}
            />
          </div>
        );
      case "select_one":
        return (
          <ConfigSelectOne
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            {...item}
          />
        );
      case "heading":
        return (
          <div key={`${i}-${item.name}`} className={`u-marginTop--40 u-marginBottom--15 ${item.hidden ? "hidden" : ""}`}>
            <h3 className="header-color field-section-header">{item.title}</h3>
          </div>
        );
      case "password":
        return (
          <ConfigInput
            key={`${i}-${item.name}`}
            handleOnChange={this.handleItemChange}
            hidden={item.hidden}
            inputType="password"
            {...item}
          />
        );
      default:
        return (
          <div key={`${i}-${item.name}`}>Unsupported config type <a href="https://help.replicated.com/docs/config-screen/config-yaml/" target="_blank" rel="noopener noreferrer">Check out our docs</a> to see all the support config types.</div>
        );
      }
    })
  }

  addLinkAttributes = () => {
    if (this.refs.markdown) {
      each(ReactDOM.findDOMNode(this.refs.markdown).getElementsByTagName("a"), (el) => {
        el.setAttribute("target", "_blank");
      });
    }
  }

  isAtLeastOneItemVisible = () => {
    const { item } = this.props;
    if (!item) return false;
    return some(this.props.item.items, (item) => {
      if (!isEmpty(item)) {
        return ConfigService.isVisible(this.props.items, item);
      }
    });
  }

  componentDidMount() {
    this.addLinkAttributes();
  }

  render() {
    const { item } = this.props;
    if (!this.isAtLeastOneItemVisible()) return null;
    return (
      <div className="flex-column flex-auto">
        {item &&
          <div id={item.name} className={`flex-auto config-item-wrapper ${this.isAtLeastOneItemVisible() ? "u-marginBottom--40" : ""}`}>
            <h3 className="header-color field-section-header">{item.title}</h3>
            {item.description !== "" ?
              <Markdown
                options={{
                  linkTarget: "_blank",
                  linkify: true,
                }}>
                {item.description}
              </Markdown>
              : null}
            <div className="config-item">
              {this.renderConfigItems(item.items)}
            </div>
          </div>
        }
      </div>
    );
  }
}
