import React from "react";
import PropTypes from "prop-types";
import { Line } from "rc-progress";

import { Utilities } from "../../utilities/utilities";
import Loader from "./Loader";

export const RENDER_PHASE = "render";

export class StepBuildingAssets extends React.Component {
  static propTypes = {
    location: PropTypes.shape({
      pathname: PropTypes.string,
    }).isRequired,
    currentRoute: PropTypes.shape({
      id: PropTypes.string,
      phase: PropTypes.string,
    }).isRequired,
    startPollingStep: PropTypes.func.isRequired,
    status: PropTypes.shape({
      type: PropTypes.string,
      detail: PropTypes.string,
    }),
  }

  componentDidMount() {
    const { startPollingStep, currentRoute } = this.props;
    if (currentRoute.phase === RENDER_PHASE) {
      startPollingStep(currentRoute.id);
    }
  }

  render() {
    /* status json looks something like

{
  "currentStep": {
    "render": {}
  },
  "phase": "render",
  "progress": {
    "source": "docker",
    "type": "json",
    "level": "info",
    "detail": "{\"id\":\"5523988621d2\",\"status\":\"Downloading\",\"image\":\"registry.replicated.com/myapp/some-image:latest\",\"progressDetail\":{\"current\":31129230,\"total\":31378422}}"
  }
}

     */
    const { status = {} } = this.props;
    const isJSON = status.type === "json";
    const parsed = isJSON ? JSON.parse(status.detail) : {};

    const message = parsed.message ? parsed.message : "";
    const isError = parsed && parsed.status === "error";
    const isSuccess = parsed && parsed.status === "success";
    const progressDetail = parsed.progressDetail ? parsed.progressDetail : null;
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
                    <span>There was an unexpected error! Please check <code className="language-bash">.ship/debug.log</code> for more details</span> : message} {progressDetail && <span> {parsed.status || "Saving"} {parsed.image} </span>}
                </p>
                {!progressDetail || percent <= 0 ? null :
                  <div className="u-marginTop--20">
                    <div className="progressBar-wrapper">
                      <Line percent={percent} strokeWidth="1" strokeColor="#337AB7" />{parsed.id}
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
