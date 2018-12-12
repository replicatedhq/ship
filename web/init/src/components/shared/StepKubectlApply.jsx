import React from "react";
import PropTypes from "prop-types";
import { Line } from "rc-progress";
import clamp from "lodash/clamp";

import { Utilities } from "../../utilities/utilities";
import Loader from "./Loader";
import StepMessage from "./StepMessage";

export const KUBECTL_PHASE = "kubectl";

export class StepKubectlApply extends React.Component {
  static propTypes = {
    location: PropTypes.shape({
      pathname: PropTypes.string,
    }).isRequired,
    currentRoute: PropTypes.shape({
      id: PropTypes.string,
      phase: PropTypes.string,
    }).isRequired,
    startPoll: PropTypes.func.isRequired,
    gotoRoute: PropTypes.func.isRequired,
    initializeStep: PropTypes.func.isRequired,
    status: PropTypes.shape({
      type: PropTypes.string,
      detail: PropTypes.string,
    }),
    startPollingStep: PropTypes.func.isRequired,
    handleAction: PropTypes.func,
  }

  constructor(props) {
    super(props);
  }

  componentDidMount() {
    const {
      currentRoute,
      startPollingStep,
    } = this.props;

    if (currentRoute.phase === KUBECTL_PHASE) {
      startPollingStep(currentRoute.id);
    }
  }

  parseStatus = () => {
    const { status = {} } = this.props;
    const { type, detail } = status;
    const isJSON = type === "json";

    const parsedDetail = isJSON ? JSON.parse(detail) : {};
    const {
      status: parsedDetailStatus,
      progressDetail,
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
        message,
      }
    }

    // TODO(Robert): for now, this is a catch all for using the progress status to determine the phase
    if (parsedDetailStatus !== "error") {
      const percent = progressDetail ? `${Utilities.calcPercent(progressDetail.current, progressDetail.total, 0)}` : 0;
      const clampedPercent = clamp(percent, 0, 100);
      return {
        isJSON,
        status: parsedDetailStatus,
        percent: clampedPercent,
        progressDetail,
        message,
      }
    }
  }

  handleAction = (action) => {
    const {
      handleAction,
      startPoll,
      currentRoute,
      gotoRoute,
    } = this.props;
    handleAction(action, false);
    startPoll(currentRoute.id, gotoRoute);
  }

  render() {
    const {
      isJSON,
      status = "",
      percent,
      progressDetail,
      message,
      actions,
    } = this.parseStatus();

    return (
      <div className="flex1 flex-column justifyContent--center alignItems--center">
        {status === "working" ?
          <div className="flex-column alignItems--center">
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
                {!message ? null :
                  <StepMessage message={message} />
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
            handleAction={this.handleAction}
          />
          : null
        }
        {status === "error" ?
          <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--normal u-textAlign--center">{message}</p>
          : null
        }
        {status === "success" ?
          <React.Fragment>
            <div className="icon progress-detail-success"></div>
            <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--normal u-textAlign--center">{message}</p>
          </React.Fragment> : null
        }
      </div>
    );
  }
}
