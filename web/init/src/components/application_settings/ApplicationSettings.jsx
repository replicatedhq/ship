import React from "react";

import ErrorBoundary from "../../ErrorBoundary";

import isEmpty from "lodash/isEmpty";

import ConfigRender from "../config_render/ConfigRender";
import Toast from "../shared/Toast";

// var EDITABLE_ITEM_TYPES = [
//   "select", "text", "textarea", "password", "file", "bool", "select_many", "select_one",
// ];

export default class ApplicationSettings extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      toastDetails: {
        opts: {}
      }
    };
  }

  componentDidUpdate(lastProps) {
    const { location, settingsFields } = this.props;
    if (this.props.settingsFields !== lastProps.settingsFields && !isEmpty(settingsFields)) {
      // this.setState({ groups: settingsFields });
      if (location.hash) {
        const el = document.getElementById(location.hash.replace("#", ""));
        if (!el) return;
        el.scrollIntoView({ behavior: "smooth", block: "start" });
      }
    }
  }

  componentDidMount() {
    if (!this.props.settingsFieldsList.length) {
      this.props.getApplicationSettings({item_values: null});
    }
  }

  // mergeItemData(groups, data) {
  //   const nextGroups = map(groups, function(group) {
  //     group.items = map(group.items, function(item) {
  //       if (item.name === data.name) {
  //         item.value = data.value;
  //         item.multi_value = data.multi_value;
  //         if (has(data, 'data')) {
  //           item.data = data.data;
  //         }
  //         if (has(data, 'multi_data')) {
  //           item.multi_data = data.multi_data;
  //         }
  //       } else {
  //         item.items = map(item.items, function(childItem) {
  //           if (childItem.name === data.name) {
  //             childItem.value = data.value;
  //             childItem.multi_value = data.multi_value;
  //             if (has(data, 'data')) {
  //               childItem.data = data.data;
  //             }
  //             if (has(data, 'multi_data')) {
  //               childItem.multi_data = data.multi_data;
  //             }
  //           }
  //           return childItem;
  //         });
  //       }
  //       return item;
  //     });
  //     return group;
  //   });
  //   return nextGroups;
  // }

  // mergeGroupItemData(groups, currentGroups) {
  //   const getItemData = (item) => {
  //     let data = {
  //       name: item.name,
  //       value: item.value,
  //       multi_value: item.multi_value,
  //     };
  //     if (has(item, 'data')) {
  //       data.data = item.data;
  //     }
  //     if (has(item, 'multi_data')) {
  //       data.multi_data = item.multi_data;
  //     }
  //     return data;
  //   };

  //   let nextGroups = _.cloneDeep(groups);
  //   forEach(currentGroups, (group) => {
  //     forEach(group.items, (item) => {
  //       if (includes(EDITABLE_ITEM_TYPES, item.type) && !item.readonly) {
  //         nextGroups = this.mergeItemData(nextGroups, getItemData(item));
  //       }
  //       forEach(item.items, (childItem) => {
  //         if (includes(EDITABLE_ITEM_TYPES, childItem.type) && !childItem.readonly) {
  //           nextGroups = this.mergeItemData(nextGroups, getItemData(childItem));
  //         }
  //       });
  //     });
  //   });
  //   return nextGroups;
  // }

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

  onConfigSaved = (data) => {
    let nextState = {};
    if (data.RunStatusText === "Starting") {
      nextState.toastDetails = {
        showToast: true,
        title: "Settings Saved",
        subText: "The app is starting.",
        type: "success",
        opts: {
          showCancelButton: true,
          cancelButtonText: "Close",
          confirmButtonText: "Take me to the Dashboard",
          confirmAction: () => { false ? () => { return; } : window.location.href = "/dashboard"; } //TODO: pass isConfirm and then should should check !isConfirm instead of false
        }
      }
    } else if (data.NextAction.State === "") {
      nextState.toastDetails = {
        showToast: true,
        title: "Settings Saved",
        subText: "You can restart the app at any time to apply these changes.",
        type: "success",
      }
    } else {
      let alertMsg = (
        `You can ${data.NextAction.Title.toLowerCase()} the app at any time to apply these changes.`
      );
      if (false) { //TODO: Should check Config.applyEnabled
        alertMsg = (
          <p>
            {alertMsg}<br />
            <i>NOTE: Applying changes may result in the application services being restarted.</i>
          </p>
        );
      }
      nextState.toastDetails = {
        showToast: true,
        title: "Settings Saved",
        type: "success",
        subText: alertMsg,
        opts: {
          showCancelButton: true,
          cancelButtonText: "Close",
          confirmButtonText: false ? "Apply Changes" : "Restart Now", //TODO: Should check Config.applyEnabled
          confirmAction: () => { true ? this.cancelToast()  : this.props.setApplicationState(data.NextAction.State) } //TODO: pass isConfirm and then should should check !isConfirm instead of false
        }
      }
    }
    this.setState(nextState);
  }

  async handleConfigSave() {
    await this.props.saveApplicationSettings(this.state.itemData, false)
      .then((response) => {
        this.onConfigSaved(response);
      })
  }

  handleConfigChange(data) {
    this.setState({ itemData: data });
    // const itemValues = map(data, partialRight(omit, "data", "multi_data"));
    // await this.props.getApplicationSettings({item_values: itemValues}, false)
    // .then((response) => {
    //   this.setState({
    //     groups: this.mergeGroupItemData(this.props.settingsFields, this.state.groups)
    //   });
    // })
    // .catch(() => {
    //   return;
    // });
  }

  render() {
    const {
      dataLoading,
      settingsFields,
      settingsFieldsList
    } = this.props;
    const { toastDetails } = this.state;

    return (
      <ErrorBoundary>
        <div className="flex-column flex1">
          <div className="flex-column flex1 u-overflow--hidden u-position--relative">
            <Toast toast={toastDetails} onCancel={this.cancelToast} />
            <div className="flex-1-auto u-overflow--auto container u-paddingTop--30">
              {dataLoading.appSettingsFieldsLoading || isEmpty(settingsFields) ?
                <p>loading</p>
                :
                <ConfigRender
                  fieldsList={settingsFieldsList}
                  fields={settingsFields}
                  handleChange={this.handleConfigChange}
                />
              }
            </div>
            <div className="flex-auto flex justifyContent--flexEnd layout-footer-actions">
              <button type="button" disabled={dataLoading.saveAppSettingsLoading} onClick={this.handleConfigSave.bind(this)} className="btn primary">{dataLoading.saveAppSettingsLoading ? "Saving" : "Save changes"}</button>
            </div>
          </div>
        </div>
      </ErrorBoundary>
    );
  }
}
