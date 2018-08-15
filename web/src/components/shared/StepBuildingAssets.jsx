import React from "react";
import autoBind from "react-autobind";
import { Utilities } from "../../utilities/utilities";
import { Line } from "rc-progress";
import Loader from "./Loader";

export default class StepBuildingAssets extends React.Component {

  constructor(props) {
    super(props);
    autoBind(this);
  }

  componentDidUpdate() {
    if (this.props.finished && this.props.location.pathname === "/render") {
      this.props.handleAction();
    }
  }

  componentDidMount() {
    this.props.startPoll();
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
