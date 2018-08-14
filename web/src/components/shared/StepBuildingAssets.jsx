import React from "react";
import autoBind from "react-autobind";
import { Utilities } from "../../utilities/utilities";
import { Line } from "rc-progress";
import Loader from "./Loader";

export default class StepBuildingAssets extends React.Component {

  constructor(props) {
    super(props);
    autoBind(this);
    this.finished = false;
  }

  componentDidMount() {
    this.startPoll();
  }

  componentDidUpdate(lastProps) {
    const { detail } = this.props.status;
    const parsedDetail = JSON.parse(detail);

    if (this.props.status !== lastProps.status) {
      clearTimeout(this.timeout);
      this.startPoll();
    }

    if(this.props.status && parsedDetail.status === "success") {
      this.finished = true;
      this.props.handleAction();
    }
  }

  isFinished() {
    return this.finished;
  }

  startPoll() {
    const self = this;
    this.timeout = setTimeout(() => {
      if (!self.finished) {
        this.props.getStep("getStatus");
      }
    }, 1000);
  }

  render() {
    const { status = {} } = this.props;
    const isJSON = status.type === "json";
    const progressDetail = isJSON ? JSON.parse(status.detail).progressDetail : null;
    let percent = progressDetail ? `${Utilities.calcPercent(progressDetail.current, progressDetail.total, 0)}` : 0;
    if (percent > 100) {
      percent = 100;

    }
    return (
      <div className="flex1 flex-column justifyContent--center alignItems--center">
        { progressDetail && progressDetail.status === "success" ?
          <div className="success">
            <span className="icon u-smallCheckWhite"></span>
          </div> :
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
                <p className="u-fontSizer--larger u-color--tundora u-fontWeight--bold u-marginTop--normal u-textAlign--center">
                  {JSON.parse(status.detail).status} {progressDetail && <span>{percent > 0 ? `${percent}%` : ""}</span>}
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
