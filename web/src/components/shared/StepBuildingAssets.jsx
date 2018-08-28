import React from "react";
import PropTypes from "prop-types";
import autoBind from "react-autobind";
import { Line } from "rc-progress";

import { Utilities } from "../../utilities/utilities";
import Loader from "./Loader";

export default class StepBuildingAssets extends React.Component {
  static propTypes = {
    location: PropTypes.shape({
      pathname: PropTypes.string,
    }).isRequired,
    routeId: PropTypes.string.isRequired,
    startPoll: PropTypes.func.isRequired,
    initializeStep: PropTypes.func.isRequired,
    status: PropTypes.shape({
      type: PropTypes.string,
      detail: PropTypes.string,
    }),
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
      initializeStep,
    } = this.props;

    if (location.pathname === "/render") {
      initializeStep(routeId);
      startPoll(routeId, gotoRoute);
    }
  }

  render() {
    const { status = {} } = this.props;
    const isJSON = status.type === "json";
    const parsed = isJSON ? JSON.parse(status.detail) : null;
    const message = parsed ? JSON.parse(status.detail).message : "";
    const isError = parsed && parsed.status === "error";
    const isSuccess = parsed && parsed.status === "success";
    const progressDetail = parsed ? JSON.parse(status.detail).progressDetail : null;
    let percent = progressDetail ? `${Utilities.calcPercent(progressDetail.current, progressDetail.total, 0)}` : 0;
    if (percent > 100) {
      percent = 100;
    }
    return (
      <div className="flex1 flex-column justifyContent--center alignItems--center StepBuildingAssets-wrapper">
        { isSuccess ?
          <div className="icon progress-detail-success"></div> :
          isError ?
            <div className="icon progress-detail-error"></div> :
            <Loader size="60" />
        }
        {status.source === "render" ?
          <div>
            <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--normal u-textAlign--center">
              { status.detail === "resolve" ? "Resolving Plan" : "Rendering Assets" }
            </p>
          </div>
          :
          <div>
            {isJSON ?
              <div>
                <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--20 u-lineHeight--more u-textAlign--center">
                  {message.length > 500 ?
                    <span>There was an unexpected error! Please check <code className="language-bash">.ship/debug.log</code> for more details</span> : message} {progressDetail && <span>{percent > 0 ? `${percent}%` : ""}</span>}
                </p>
                {!progressDetail ? null :
                  <div className="u-marginTop--20">
                    <div className="progressBar-wrapper">
                      <Line percent={percent} strokeWidth="1" strokeColor="#337AB7" />
                    </div>
                  </div>
                }
              </div>
              :
              <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--normal u-textAlign--center">{status.detail}</p>
            }
          </div>
        }
      </div>
    );
  }
}
