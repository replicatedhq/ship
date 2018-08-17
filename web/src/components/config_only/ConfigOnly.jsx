import React from "react";
import ErrorBoundary from "../../ErrorBoundary";
import autoBind from "react-autobind";

import isEmpty from "lodash/isEmpty";
import map from "lodash/map";
import partialRight from "lodash/partialRight";
import omit from "lodash/omit";
import has from "lodash/has";
import forEach from "lodash/forEach";
import includes from "lodash/includes";
import _ from "lodash";

import Layout from "../../Layout";
import ConfigRender from "../config_render/ConfigRender";
import Toast from "../shared/Toast";
import Loader from "../shared/Loader";

const EDITABLE_ITEM_TYPES = [
  "select", "text", "textarea", "password", "file", "bool", "select_many", "select_one",
];

export default class ConfigOnly extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      toastDetails: {
        opts: {}
      }
    };
    autoBind(this);
  }

  componentDidMount() {
    this.props.getChannel();
    if (!this.props.settingsFieldsList.length) {
      this.props.getApplicationSettings({item_values: null});
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.phase !== lastProps.phase) {
      if (this.props.phase !== "render.config") {
        this.props.history.push("/");
      }
    }
  }

  mergeItemData(groups, data) {
    const nextGroups = map(groups, function(group) {
      group.items = map(group.items, function(item) {
        if (item.name === data.name) {
          item.value = data.value;
          item.multi_value = data.multi_value;
          if (has(data, "data")) {
            item.data = data.data;
          }
          if (has(data, "multi_data")) {
            item.multi_data = data.multi_data;
          }
        } else {
          item.items = map(item.items, function(childItem) {
            if (childItem.name === data.name) {
              childItem.value = data.value;
              childItem.multi_value = data.multi_value;
              if (has(data, "data")) {
                childItem.data = data.data;
              }
              if (has(data, "multi_data")) {
                childItem.multi_data = data.multi_data;
              }
            }
            return childItem;
          });
        }
        return item;
      });
      return group;
    });
    return nextGroups;
  }

  mergeGroupItemData(groups, currentGroups) {
    const getItemData = (item) => {
      let data = {
        name: item.name,
        value: item.value,
        multi_value: item.multi_value,
      };
      if (has(item, "data")) {
        data.data = item.data;
      }
      if (has(item, "multi_data")) {
        data.multi_data = item.multi_data;
      }
      return data;
    };

    let nextGroups = _.cloneDeep(groups);
    forEach(currentGroups, (group) => {
      forEach(group.items, (item) => {
        if (includes(EDITABLE_ITEM_TYPES, item.type) && !item.readonly) {
          nextGroups = this.mergeItemData(nextGroups, getItemData(item));
        }
        forEach(item.items, (childItem) => {
          if (includes(EDITABLE_ITEM_TYPES, childItem.type) && !childItem.readonly) {
            nextGroups = this.mergeItemData(nextGroups, getItemData(childItem));
          }
        });
      });
    });
    return nextGroups;
  }

  cancelToast() {
    let nextState = {};
    nextState.toastDetails = {
      showToast: false,
      title: "",
      subText: "",
      type: "default",
      opts: {}
    };
    this.setState(nextState)
  }

  onConfigSaved() {
    const {
      actions,
      handleAction,
      finalizeApplicationSettings,
    } = this.props;
    const configAction = actions[0];
    const { text } = configAction;

    const nextState = {
      toastDetails: {
        showToast: true,
        title: "All changes have been saved.",
        type: "default",
        opts: {
          showCancelButton: true,
          confirmButtonText: text,
          confirmAction: async () => {
            await finalizeApplicationSettings(this.state.itemData, false).catch();
            await handleAction(configAction, true);
          },
        },
      },
    };

    this.setState(nextState);
  }

  onConfigError() {
    const { configErrors } = this.props;
    if (!configErrors.length) return;
    configErrors.map((err) => {
      const el = document.getElementById(`${err.fieldName}-errblock`);
      el.innerHTML = err.message;
      el.classList.add("visible");
    })
  }

  async handleConfigSave() {
    let errs = document.getElementsByClassName("config-errblock");
    for (let i = 0; i < errs.length; i++) {
      errs[i].classList.remove("visible");
    }
    await this.props.saveApplicationSettings(this.state.itemData, false)
      .then((response) => {
        if (!response) {
          this.onConfigError();
          return;
        }
        this.onConfigSaved();
      })
      .catch()
  }

  async pingServer(data) {
    const itemValues = map(data, partialRight(omit, "data", "multi_data"));
    await this.props.getApplicationSettings({item_values: itemValues}, false)
      .then()
      .catch();
  }

  handleConfigChange(data) {
    this.setState({ itemData: data });
    this.pingServer(data);
  }

  render() {
    const {
      dataLoading,
      settingsFields,
      settingsFieldsList
    } = this.props;
    const { toastDetails } = this.state;

    return (
      <Layout configOnly={true}>
        <ErrorBoundary>
          <div className="flex-column flex1">
            <div className="flex-column flex1 u-overflow--hidden u-position--relative">
              <Toast toast={toastDetails} onCancel={this.cancelToast} />
              <div className="flex-1-auto flex-column u-overflow--auto container u-paddingTop--30">
                {dataLoading.appSettingsFieldsLoading || isEmpty(settingsFields) ?
                  <div className="flex1 flex-column justifyContent--center alignItems--center">
                    <Loader size="60" />
                  </div>
                  :
                  <ConfigRender
                    fieldsList={settingsFieldsList}
                    fields={settingsFields}
                    handleChange={this.handleConfigChange}
                  />
                }
              </div>
              <div className="flex-auto flex justifyContent--flexEnd layout-footer-actions">
                <button type="button" disabled={dataLoading.saveAppSettingsLoading} onClick={this.handleConfigSave} className="btn primary">{dataLoading.saveAppSettingsLoading ? "Saving" : "Save changes"}</button>
              </div>
            </div>
          </div>
        </ErrorBoundary>
      </Layout>
    );
  }
}
