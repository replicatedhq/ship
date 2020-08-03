import React from "react";
import ErrorBoundary from "../../ErrorBoundary";
import each from 'lodash/each';
import isEmpty from 'lodash/isEmpty';
import map from 'lodash/map';
import partialRight from 'lodash/partialRight';
import omit from 'lodash/omit';

import { ConfigService } from "../../services/ConfigService";

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
      },
    };
  }

  componentDidMount() {
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
    if (this.props.settingsFields !== lastProps.settingsFields) {
      const data = this.getData(this.props.settingsFields);
      this.setState({ itemData: data });
    }
  }

  getData = (groups) => {
    const getItemData = (item) => {
      let data = {
        name: item.name,
        value: item.value,
        multi_value: item.multi_value,
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
    each(groups, (group) => {
      if (ConfigService.isEnabled(groups, group)) {
        each(group.items, (item) => {
          if (ConfigService.isEnabled(groups, item)) {
            if (item.type !== "select_many") {
              data.push(getItemData(item));
            }
            if (item.type !== "select_one") {
              each(item.items, (childItem) => {
                data.push(getItemData(childItem));
              });
            }
          }
        });
      }
    });
    return data;
  }

  cancelToast = () => {
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

  onConfigSaved = () => {
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

  onConfigError = () => {
    const { configErrors } = this.props;
    if (!configErrors.length) return;
    configErrors.map((err) => {
      // TODO: Refactor to pass errors down instead of manipulating
      // the DOM
      const el = document.getElementById(`${err.fieldName}-errblock`);
      if (el) {
        el.innerHTML = err.message;
        el.classList.add("visible");
      }
    })
  }

  handleConfigSave = async (e, nextStep = false) => {
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
        if(nextStep) this.state.toastDetails.opts.confirmAction();
      })
      .catch()
  }

  pingServer = async (data) => {
    const itemValues = map(data, partialRight(omit, "data", "multi_data"));
    await this.props.getApplicationSettings({item_values: itemValues}, false)
      .then()
      .catch();
  }

  handleConfigChange = (data) => {
    this.setState({ itemData: data });
    this.pingServer(data);
  }

  render() {
    const {
      dataLoading,
      settingsFields,
      settingsFieldsList,
      routeId,
      goBack,
      firstRoute
    } = this.props;
    const { toastDetails } = this.state;

    return (
      <Layout configOnly={true} configRouteId={routeId}>
        <ErrorBoundary className="flex-column">
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
                    getData={this.getData}
                  />
                }
              </div>
              <div className="flex-auto flex layout-footer-actions">
                <div className="flex flex1">
                  {firstRoute ? null :
                    <div className="flex-auto u-marginRight--normal">
                      <button className="btn secondary" onClick={() => goBack()}>Back</button>
                    </div>
                  }
                  <div className="flex-column flex-verticalCenter">
                    <p className="u-margin--none u-marginRight--30 u-fontSize--small u-color--dustyGray u-fontWeight--normal">Contributed by <a target="_blank" rel="noopener noreferrer" href="https://replicated.com" className="u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover">Replicated</a></p>
                  </div>
                </div>
                <div className="flex flex1 justifyContent--flexEnd">
                  <button type="button" disabled={dataLoading.saveAppSettingsLoading} onClick={this.handleConfigSave} className="btn secondary u-marginRight--10">{dataLoading.saveAppSettingsLoading ? "Saving" : "Save changes"}</button>
                  <button type="button" disabled={dataLoading.saveAppSettingsLoading} onClick={(e) => this.handleConfigSave(e, true)} className="btn primary">Save and continue to next step</button>
                </div>
              </div>
            </div>
          </div>
        </ErrorBoundary>
      </Layout>
    );
  }
}
