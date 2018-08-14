import React from "react";
import autoBind from "react-autobind";
import { Utilities } from "../../utilities/utilities";
import { Line } from "rc-progress";
import Loader from "./Loader";
import RenderActions from "./RenderActions";

export default class StepBuildingAssets extends React.Component {
  
  constructor(props) {
    super(props);
    autoBind(this);
    this.finished = false;
  }
  
  componentDidUpdate(lastProps) {
    if (this.props.status !== lastProps.status) {
      clearTimeout(this.timeout);
      this.startPoll();
    }
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
    const actions = this.props.actions || null;
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
            <div className="u-marginTop--30 flex justifyContent--flexEnd">
              <RenderActions actions={actions} handleAction={this.props.handleAction} isLoading={this.props.isLoading} />
            </div>
          </div>
        }
      </div>
    );
  }
}
