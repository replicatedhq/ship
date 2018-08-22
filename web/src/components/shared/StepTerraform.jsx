import React from "react";
import PropTypes from "prop-types";
import autoBind from "react-autobind";
import { Line } from "rc-progress";
import clamp from "lodash/clamp";

import { Utilities } from "../../utilities/utilities";
import Loader from "./Loader";
import StepMessage from "./StepMessage";

export default class StepPreparingTerraform extends React.Component {
  static propTypes = {
    location: PropTypes.shape({
      pathname: PropTypes.string,
    }).isRequired,
    routeId: PropTypes.string.isRequired,
    startPoll: PropTypes.func.isRequired,
    status: PropTypes.shape({
      type: PropTypes.string,
      detail: PropTypes.string,
    }).isRequired,
    handleAction: PropTypes.func,
  }

  constructor(props) {
    super(props);
    autoBind(this);
  }

  componentDidMount() {
    const {
      startPoll,
      routeId,
      gotoRoute,
      location,
    } = this.props;

    if (location.pathname === "/terraform") {
      startPoll(routeId, gotoRoute);
    }
  }

  parseStatus() {
    const { status = {} } = this.props;
    const { type, detail } = status;
    const isJSON = type === "json";

    const parsedDetail = isJSON ? JSON.parse(detail) : {};
    const {
      status: parsedDetailStatus,
      progressDetail,
      error,
      message,
      actions,
    } = parsedDetail;

    if (parsedDetailStatus === "message") {
      return {
        actions,
        isJSON,
        status: parsedDetailStatus,
        message,
      }
    }

    if (parsedDetailStatus === "error") {
      return {
        isJSON,
        status: parsedDetailStatus,
        error,
      }
    }

    if (parsedDetailStatus !== "error") {
      const percent = progressDetail ? `${Utilities.calcPercent(progressDetail.current, progressDetail.total, 0)}` : 0;
      const clampedPercent = clamp(percent, 0, 100);
      return {
        isJSON,
        status: parsedDetailStatus,
        percent: clampedPercent,
        progressDetail,
      }
    }
  }

  render() {
    const {
      isJSON,
      status = "",
      percent,
      progressDetail,
      message,
      actions,
      error,
    } = this.parseStatus();
    const { handleAction } = this.props;

    return (
      <div className="flex1 flex-column justifyContent--center alignItems--center">
        {status === "working" ?
          <div>
            <Loader size="60" />
            {isJSON ?
              <div>
                {!progressDetail ? null :
                  <div className="u-marginTop--20">
                    <div className="progressBar-wrapper">
                      <Line percent={percent} strokeWidth="1" strokeColor="#337AB7" />
                    </div>
                  </div>
                }
              </div>
              :
              <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--normal u-textAlign--center">{status}</p>
            }
          </div>: null
        }
        {status === "message" ?
          <StepMessage
            message={message}
            actions={actions}
            handleAction={handleAction}
          />
          : null
        }
        {status === "error" ?
          <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--normal u-textAlign--center">{error}</p>
          : null
        }
      </div>
    );
  }
}
