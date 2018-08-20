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
    const settingsFields = keyBy(orderedFields, "name");

    const appSidebarSubItems = map(settingsFields, (field) => {
      if (!isAtLeastOneItemVisible(field)) return;
      const { title, name } = field;
      const label = title === "" ?  Utilities.toTitleCase(name.replace("-", " ")) : title;

      return {
        id: name,
        label,
      };
    });

    return Object.assign({}, state, {
      settingsFields,
      settingsFieldsList: map(settingsFields, "name"),
      version: action.payload.Version,
      appSidebarSubItems,
    });
  case constants.SET_CONFIG_ERRORS:
    const errors = Object.assign({}, action.payload);

    const errorsArr = map(errors, (error) => {
      const { message, name} = error;
      return {
        message,
        fieldName: name,
      };
    });

    return Object.assign({}, state, {
      configErrors: errorsArr
    });
  default:
    return state;
  }
}
