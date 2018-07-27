import React from "react";
import autoBind from "react-autobind";
import { Utilities } from "../../utilities/utilities";
import { Line } from "rc-progress";
import Loader from "./Loader";

export default class StepPreparingTerraform extends React.Component {

  constructor(props) {
    super(props);
    autoBind(this);
  }

  componentDidMount() {
    this.startPoll();
  }

  componentDidUpdate(lastProps) {
    if (this.props.status !== lastProps.status) {
      clearTimeout(this.timeout);
      this.startPoll();
    }
  }

  componentWillUnmount() {
    clearTimeout(this.timeout);
  }

  startPoll() {
    this.timeout = setTimeout(() => this.props.getStep("getStatus"), 1000);
  }

  render() {
    const { status } = this.props;
    const isJSON = status.type === "json";
    const progressDetail = isJSON ? JSON.parse(status.detail).progressDetail : null;
    const percent = progressDetail ? `${Utilities.calcPercent(progressDetail.current, progressDetail.total, 0)}` : 0;
    return (
      <div className="flex1 flex-column justifyContent--center alignItems--center">
        <Loader size="60" />
        {status.source === "render" ? null :
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
