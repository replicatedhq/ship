import React from "react";
import { withRouter } from "react-router-dom";
import autoBind from "react-autobind";
import find from "lodash/find";
import findIndex from "lodash/findIndex";
import indexOf from "lodash/indexOf";

import Loader from "./Loader";
import StepMessage from "./StepMessage";
import StepDone from "./StepDone";
import StepBuildingAssets from "./StepBuildingAssets";
import StepHelmIntro from "../../containers/HelmChartInfo";
import StepHelmValues from "../kustomize/HelmValuesEditor";
import KustomizeEmpty from "../kustomize/kustomize_overlay/KustomizeEmpty";
import KustomizeOverlay from "../../containers/KustomizeOverlay";

import "../../scss/components/shared/DetermineStep.scss";

class DetermineComponentForRoute extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      maxPollReached: false,
    };
    autoBind(this);
  }

  componentDidMount() {
    this.props.getContentForStep(this.props.routeId);
  }

  componentDidUpdate() {
    const { progress = {} } = this.props;
    const { detail = "{}" } = progress;
    const parsedDetail = JSON.parse(detail);

    const pollStartedButNotFinished = !this.state.finished && this.state.startPoll;
    if(parsedDetail.status === "success" && pollStartedButNotFinished) {
      this.setState({ finished: true, startPoll: false }, () => {
        this.stopPoll();
      });
    }
  }

  async handleAction(action, gotoNext) {
    await this.props.finalizeStep({action});
    if (gotoNext) {
      this.gotoRoute();
    }
  }

  gotoRoute(route) {
    let nextRoute = route;

    if (!nextRoute) {
      const currRoute = find(this.props.routes, ["id", this.props.routeId]);
      const currIndex = indexOf(this.props.routes, currRoute);
      nextRoute = this.props.routes[currIndex + 1];
    }
    this.props.history.push(`/${nextRoute.id}`);
  }

  async getCurrentStep(stepId) {
    const apiEndpoint = window.env.API_ENDPOINT;
    const url = `${apiEndpoint}/navcycle/step/${stepId}`;
    const response = await fetch(url, {
      method: "GET",
      headers: {
        "Accept": "application/json",
      },
    });
    const body = await response.json();
    return body;
  }

  async skipKustomize() {
    const kustomizeStepIndex = findIndex(this.props.routes, { phase: "kustomize" });
    const kustomizeStep = this.props.routes[kustomizeStepIndex];
    const stepAfterKustomize = this.props.routes[kustomizeStepIndex + 1];

    const { actions } = await this.getCurrentStep(kustomizeStep.id);
    this.handleAction(actions[0]);
    this.startPoll(kustomizeStep.id, () => this.gotoRoute(stepAfterKustomize));
  }

  async startPoll(routeId, cb) {
    let finished = false;

    if (!this.interval) {
      this.interval = setInterval(() => {
        if (finished) {
          clearInterval(this.interval);
          cb();
        } else {
          this.getCurrentStep(routeId).then(({ progress }) => {
            const { detail } = progress;
            const parsedDetail = JSON.parse(detail);
            finished = parsedDetail.status === "success";
          });
        }
      }, 1000);
    }
  }

  stopPoll() {
    clearInterval(this.interval);
  }

  renderStep(phase) {
    const { currentStep, progress, actions, location } = this.props;
    if (!phase || !phase.length) return null;
    switch (phase) {
    case "requirementNotMet":
      return (
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <p className="u-fontSize--large u-fontWeight--medium u-color--tundora u-marginBottom--20">Whoa there, you're getting a little ahead of yourself. There are steps that need to be completed before you can be here.</p>
          <button className="btn primary" onClick={() => { this.props.history.push(`/${this.props.routes[0].id}`)}}>Take me back</button>
        </div>
      )
    case "message":
      return (
        <StepMessage
          actions={actions}
          message={currentStep.message}
          level={currentStep.level}
          handleAction={this.handleAction}
          isLoading={this.props.dataLoading.submitActionLoading}
        />
      );
    case "stream":
      return (
        <StepMessage
          actions={actions}
          message={currentStep.message}
          level={currentStep.level}
          handleAction={this.handleAction}
          isLoading={this.props.dataLoading.submitActionLoading || !currentStep.message.contents}
        />
      );
    case "render":
      return (
        <StepBuildingAssets
          startPoll={() => this.startPoll(this.props.routeId, this.gotoRoute)}
          routeId={this.props.routeId}
          finished={this.state.finished}
          handleAction={this.handleAction}
          location={location}
          status={progress || currentStep.status}
        />
      );
    case "terraform.prepare":
      return (
        <StepBuildingAssets
          stepId={this.props.routeId}
        />
      );
    case "helm-intro":
      return (
        <StepHelmIntro
          actions={actions}
          helmChartMetadata={this.props.helmChartMetadata}
          handleAction={this.handleAction}
          isLoading={this.props.dataLoading.submitActionLoading}
        />
      );
    case "helm-values":
      return (
        <StepHelmValues
          saveValues={this.props.saveHelmChartValues}
          getStep={currentStep.helmValues}
          isNewRouter={this.props.isNewRouter}
          helmChartMetadata={this.props.helmChartMetadata}
          actions={actions}
          handleAction={this.handleAction}
          isLoading={this.props.dataLoading.submitActionLoading}
        />
      );
    case "kustomize-intro":
      return (
        <KustomizeEmpty
          actions={actions}
          handleAction={this.handleAction}
          skipKustomize={this.skipKustomize}
        />
      );
    case "kustomize":
      return (
        <KustomizeOverlay
          startPoll={() => this.startPoll(this.props.routeId, this.gotoRoute)}
          finished={this.state.finished}
          location={location}
          actions={actions}
          isNavcycle={true}
          finalizeStep={this.props.finalizeStep}
          handleAction={this.handleAction}
          currentStep={currentStep}
          dataLoading={this.props.dataLoading}
        />
      );
    case "done":
      return (
        <StepDone />
      );
    default:
      return (
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <Loader size="60" />
        </div>
      );
    }
  }

  render() {
    const { phase, dataLoading } = this.props;
    const isLoadingStep = phase === "loading";
    return (
      <div className="flex-column flex1">
        <div className="flex-column flex1 u-overflow--hidden u-position--relative">
          <div className="flex-1-auto flex-column u-overflow--auto container u-paddingTop--30">
            {(isLoadingStep || dataLoading.getCurrentStepLoading || dataLoading.getHelmChartMetadataLoading) && !this.state.maxPollReached ?
              <div className="flex1 flex-column justifyContent--center alignItems--center">
                <Loader size="60" />
              </div>
              : this.state.maxPollReached ?
                <div className="flex1 flex-column justifyContent--center alignItems--center">
                  <p className="u-fontSize--large u-fontWeight--medium u-color--tundora">Oops, something isn't quite right. If you continue to experience this problem contact <a href="mailto:support@replicated.com">support@replicated.com</a></p>
                </div>
                :
                this.renderStep(phase)
            }
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(DetermineComponentForRoute)