import React from "react";
import ReactDOM from "react-dom";
import PropTypes from "prop-types";
import { Line } from "rc-progress";
import clamp from "lodash/clamp";

import { Utilities } from "../../utilities/utilities";
import Loader from "./Loader";
import StepMessage from "./StepMessage";

export const TERRAFORM_PHASE = "terraform";

export class StepTerraform extends React.Component {
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
    handleAction: PropTypes.func,
    startPollingStep: PropTypes.func.isRequired,
  }

  componentDidMount() {
    const {
      startPollingStep,
      currentRoute,
    } = this.props;

    if (currentRoute.phase === TERRAFORM_PHASE) {
      startPollingStep(currentRoute.id);
    }
  }

  componentDidUpdate() {
    this.scrollToLogsBottom();
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

  scrollToLogsBottom = (elm) => {
    const {
      status = "",
    } = this.parseStatus();
    const node = ReactDOM.findDOMNode(this);
    const child = node.querySelector('.term-container');
    if(child) {
      const height = child.scrollHeight;
      child.scrollTo({ top: height, behavior: "instant" });
    }
  }

  render() {
    const {
      isJSON,
      percent,
      status,
      message,
      progressDetail,
      actions,
    } = this.parseStatus();

    return (
      <div className="flex1 flex flex-column justifyContent--center">
        {status === "working" ?
          <div className="flex flex1 flex-column u-paddingTop--30 justifyContent--center">
            <div className="flex justifyContent--center">
              <Loader size="60" />
            </div>
            {isJSON ?
              <div className="flex flex-column">
                {!progressDetail ? null :
                  <div className="flex flex1 flex-column">
                    <div className="u-marginTop--20">
                      <div className="progressBar-wrapper">
                        <Line percent={percent} strokeWidth="1" strokeColor="#337AB7" />
                      </div>
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
            setLogsRef={this.setLogsRef}
          />
          : null
        }
        {status === "error" ?
          <div className="Error--wrapper flex flex-column alignItems--center">
            <div className="icon progress-detail-error"></div>
            <p className="u-fontSizer--larger u-color--tundora u-lineHeight--normal u-fontWeight--bold u-marginTop--normal u-textAlign--center">{message}</p>
          </div>
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
