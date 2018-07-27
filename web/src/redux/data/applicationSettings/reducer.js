import { constants } from "./actions";
import keyBy from "lodash/keyBy";
import sortBy from "lodash/sortBy";
import map from "lodash/map";
import some from "lodash/some";
import isEmpty from "lodash/isEmpty";
import { Utilities } from "../../../utilities/utilities";
import { ConfigService } from "../../../services/ConfigService";

const applicationSettingsState = {
  settingsFields: {},
  settingsFieldsList: [],
  appSidebarSubItems: [],
  configErrors: [],
  version: null
};

function isAtLeastOneItemVisible(field) {
  return some(field.items, (item) => {
    if (!isEmpty(item)) {
      return ConfigService.isVisible(field.items, item);
    }
  });
}

export function settingsData(state = applicationSettingsState, action) {
  switch (action.type) {
  case constants.RECEIVE_APPLICATION_SETTINGS:
    const resBody = Object.assign({}, action.payload.Groups);
    const orderedFields = sortBy(resBody, "position");
    const fields = keyBy(orderedFields, "name");

    let subItemsArr = [];
    map(fields, (field) => {
      if (!isAtLeastOneItemVisible(field)) return;
      const label = field.title === "" ?  Utilities.toTitleCase(field.name.replace("-", " ")) : field.title;
      const obj = {
        id: field.name,
        label: label
      }
      subItemsArr.push(obj);
    });

    return Object.assign({}, state, {
      settingsFields: fields,
      settingsFieldsList: map(fields, "name"),
      version: action.payload.Version,
      appSidebarSubItems: subItemsArr
    });
  case constants.SET_CONFIG_ERRORS:
    const errors = Object.assign({}, action.payload);
    let errorsArr = [];
    map(errors, (error) => {
      const errObj = {
        message: error.message,
        fieldName: error.name
      };
      errorsArr.push(errObj);
    });

    return Object.assign({}, state, {
      configErrors: errorsArr
    });
  default:
    return state;
  }
}
