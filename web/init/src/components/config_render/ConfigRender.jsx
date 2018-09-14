import React from "react";
import autoBind from "react-autobind";
import keyBy from "lodash/keyBy";
import _ from "lodash";

import ConfigGroups from "./ConfigGroups";
import { ConfigService } from "../../services/ConfigService";

export default class ConfigRender extends React.Component {

  constructor(props) {
    super(props);
    this.state= {
      groups: this.props.fields
    }
    autoBind(this);
  }

  triggerChange(data) {
    if (this.props.handleChange) {
      this.props.handleChange(data);
    }
  }

  getData(groups) {
    const getItemData = (item) => {
      let data = {
        name: item.name,
        value: "",
        multi_value: [],
      };
      if (item.multiple) {
        if (item.multi_value && item.multi_value.length) {
          data.multi_value = item.multi_value;
        } else if (item.default) {
          data.multi_value = [item.default];
        }
      } else {
        if (item.value && item.value.length) {
          data.value = item.value;
        } else {
          data.value = item.default;
        }
      }
      if (item.type === "file") {
        data.data = "";
        if (item.multiple) {
          if (item.multi_data && item.multi_data.length) {
            data.multi_data = item.multi_data;
          } else {
            data.multi_data = [];
          }
        } else {
          data.data = item.data;
        }
      }
      return data;
    };

    let data = [];
    _.each(groups, (group) => {
      if (ConfigService.isEnabled(groups, group)) {
        _.each(group.items, (item) => {
          if (ConfigService.isEnabled(groups, item)) {
            if (item.type !== "select_many") {
              data.push(getItemData(item));
            }
            if (item.type !== "select_one") {
              _.each(item.items, (childItem) => {
                data.push(getItemData(childItem));
              });
            }
          }
        });
      }
    });
    return data;
  }

  handleGroupsChange(groupName, itemName, value, data) {
    const getValues = (val) => {
      if (!val) {
        return [];
      }
      return _.isArray(val) ? val : [val];
    };
    const getValue = (val) => {
      if (!val) {
        return val;
      }
      return _.isArray(val) ? _.first(val) : val;
    };
    let groups = _.map(this.props.fields, (group) => {
      if (group.name === groupName) {
        group.items = _.map(group.items, (item) => {
          if (!(item.type in ["select_many", "label", "heading"]) && item.name === itemName) {
            if (item.multiple) {
              item.multi_value = getValues(value);
              if (item.type === "file") {
                item.multi_data = getValues(data);
              }
            } else {
              item.value = getValue(value);
              if (item.type === "file") {
                item.data = getValue(data);
              }
            }
          } else {
            if (item.type !== "select_one") {
              item.items = _.map(item.items, (childItem) => {
                if (childItem.name === itemName) {
                  if (childItem.multiple) {
                    childItem.multi_value = getValues(value);
                    if (childItem.type === "file") {
                      childItem.multi_data = getValues(data);
                    }
                  } else {
                    childItem.value = getValue(value);
                    if (childItem.type === "file") {
                      childItem.data = getValue(data);
                    }
                  }
                }
                return childItem;
              });
            }
          }
          return item;
        });
      }
      return group;
    });

    this.setState({groups: keyBy(groups, "name")});

    // TODO: maybe this should only be on submit
    this.triggerChange(this.getData(groups));
  }

  componentDidUpdate(lastProps) {
    if (this.props.fields !== lastProps.fields) {
      this.setState({
        groups: keyBy(ConfigService.filterGroups(this.props.fields, this.props.filters), "name"),
      });
    }
  }

  render() {
    const { fieldsList } = this.props;
    return (
      <div className="flex-column flex1">
        <ConfigGroups
          fieldsList={fieldsList}
          fields={this.state.groups}
          handleChange={this.handleGroupsChange}
        />
      </div>
    );
  }
}
