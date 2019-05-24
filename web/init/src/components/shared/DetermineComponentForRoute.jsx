import React from "react";
import PropTypes from "prop-types";
import { withRouter } from "react-router-dom";
import find from "lodash/find";
import indexOf from "lodash/indexOf";

import Loader from "./Loader";
import StepMessage from "./StepMessage";
import { RENDER_PHASE, StepBuildingAssets } from "./StepBuildingAssets";
import StepHelmIntro from "../../containers/HelmChartInfo";
import StepHelmValues from "../kustomize/HelmValuesEditor";
import { TERRAFORM_PHASE, StepTerraform } from "./StepTerraform";
import { KUBECTL_PHASE, StepKubectlApply } from "./StepKubectlApply";
import KustomizeEmpty from "../kustomize/kustomize_overlay/KustomizeEmpty";
import KustomizeOverlay from "../../containers/KustomizeOverlay";
import ConfigOnly from "../../containers/ConfigOnly";
import { fetchContentForStep } from "../../redux/data/appRoutes/actions";

export class DetermineComponentForRoute extends React.Component {
  static propTypes = {
    /** Callback function to be invoked at the finalization of the Ship Init flow */
    onCompletion: PropTypes.func,
  }

  constructor(props) {
    super(props);
    this.state = {
      maxPollReached: false,
    };
  }

  componentDidMount() {
    this.getContentForStep();
  }

  handleAction = async (action, gotoNext) => {
    await this.props.finalizeStep({action});
    if (gotoNext) {
      this.gotoRoute();
    }
  }

  getContentForStep = () => {
    const { getContentForStep, currentRoute } = this.props;
    const { id: routeId } = currentRoute;
    getContentForStep(routeId);
  }

  gotoRoute = async(route) => {
    let nextRoute = route;
    const { basePath, routes, currentRoute, history, onCompletion } = this.props;

    if (!nextRoute) {
      const currRoute = find(routes, ["id", currentRoute.id]);
      const currIndex = indexOf(routes, currRoute);
      nextRoute = routes[currIndex + 1];
    }

    if (!nextRoute) {
      if (onCompletion) {
        await this.handleShutdown();
        return onCompletion();
      }

      await this.handleShutdown();
      return this.gotoDone();
    }

    history.push(`${basePath}/${nextRoute.id}`);
  }

  handleShutdown = async () => {
    const { apiEndpoint, shutdownApp } = this.props;

    const url = `${apiEndpoint}/shutdown`;
    await fetch(url, {
      method: "POST",
      headers: {
        "Accept": "application/json",
      },
    });
    await shutdownApp();
  }

  gotoDone = () => {
    const { basePath } = this.props;
    this.props.history.push(`${basePath}/done`);
  }

  skipKustomize = async () => {
    const { apiEndpoint } = this.props;
    const { actions, progress } = await fetchContentForStep(apiEndpoint, "kustomize");

    let stepValid = true;
    let kustomizeErrorMessage = "";
    if (progress && progress.detail) {
      const { status, message } = JSON.parse(progress.detail);
      stepValid = status !== "error";
      kustomizeErrorMessage = message;
    }

    if (stepValid) {
      const [finalizeKustomize] = actions;
      await this.props.finalizeStep({ action: finalizeKustomize });

      this.startPoll("kustomize", this.gotoRoute);
    } else {
      // TODO: Handle case where error detected in Kustomize step on skip
      //       This can occur even if no overlays created since kustomize
      //       build is always executed.
      alert(`Error detected, cancelling kustomize step skip. Error below: \n${kustomizeErrorMessage}`);
    }
  }

  startPoll = async (routeId, cb) => {
    if (!this.props.isPolling) {
      this.props.pollContentForStep(routeId, cb);
    }
  }

  startPollingStep = (routeId) => {
    const { initializeStep } = this.props;
    initializeStep(routeId);
    this.startPoll(routeId, () => {
      // Timeout to wait a little bit before transitioning to the next step
      setTimeout(this.gotoRoute, 500);
    });
  }

  renderStep = () => {
    const {
      currentStep = {},
      progress,
      actions,
      location,
      initializeStep,
      phase,
      currentRoute,
      routes
    } = this.props;
    const routeId = currentRoute.id;
    const firstRouteIdx = indexOf(routes, find(routes, ["id", currentRoute.id]));
    const firstRoute = firstRouteIdx === 0;

    if (!phase || !phase.length) return null;
    switch (phase) {
    case "requirementNotMet":
      return (
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <p className="u-fontSize--large u-fontWeight--medium u-color--tundora u-marginBottom--20">Whoa there, you're getting a little ahead of yourself. There are steps that need to be completed before you can be here.</p>
          <button className="btn primary" onClick={this.props.history.goBack}>Take me back</button>
        </div>
      )
    case "message":
      return (
        <StepMessage
          actions={actions}
          message={currentStep.message}
          level={currentStep.level}
          handleAction={this.handleAction}
          firstRoute={firstRoute}
          goBack={this.props.history.goBack}
          isLoading={this.props.dataLoading.submitActionLoading}
        />
      );
    case "config":
      return (
        <ConfigOnly
          actions={actions}
          handleAction={this.handleAction}
          routeId={routeId}
          firstRoute={firstRoute}
          goBack={this.props.history.goBack}
        />
      );
    case "stream":
      return (
        <StepMessage
          actions={actions}
          message={currentStep.message}
          level={currentStep.level}
          handleAction={this.handleAction}
          goBack={this.props.history.goBack}
          firstRoute={firstRoute}
          isLoading={this.props.dataLoading.submitActionLoading || !currentStep.message.contents}
        />
      );
    case RENDER_PHASE:
      return (
        <StepBuildingAssets
          startPollingStep={this.startPollingStep}
          location={location}
          status={progress || currentStep.status}
          currentRoute={currentRoute}
        />
      );
    case TERRAFORM_PHASE:
      return (
        <StepTerraform
          startPollingStep={this.startPollingStep}
          currentRoute={currentRoute}
          startPoll={this.startPoll}
          location={location}
          status={progress || currentStep.status}
          handleAction={this.handleAction}
          gotoRoute={this.gotoRoute}
          goBack={this.props.history.goBack}
          firstRoute={firstRoute}
          initializeStep={initializeStep}
        />
      );
    case KUBECTL_PHASE:
      return (
        <StepKubectlApply
          startPollingStep={this.startPollingStep}
          currentRoute={currentRoute}
          startPoll={this.startPoll}
          location={location}
          status={progress || currentStep.status}
          handleAction={this.handleAction}
          gotoRoute={this.gotoRoute}
          initializeStep={initializeStep}
        />
      );
    case "helm-intro":
      return (
        <StepHelmIntro
          actions={actions}
          isUpdate={currentStep.helmIntro.isUpdate}
          shipAppMetadata={this.props.shipAppMetadata}
          handleAction={this.handleAction}
          goBack={this.props.history.goBack}
          firstRoute={firstRoute}
          isLoading={this.props.dataLoading.submitActionLoading}
        />
      );
    case "helm-values":
      return (
        <StepHelmValues
          saveValues={this.props.saveHelmChartValues}
          getStep={currentStep.helmValues}
          shipAppMetadata={this.props.shipAppMetadata}
          actions={actions}
          handleAction={this.handleAction}
          goBack={this.props.history.goBack}
          firstRoute={firstRoute}
          isLoading={this.props.dataLoading.submitActionLoading}
        />
    );
    case "kustomize-intro":
      return (
        <KustomizeEmpty
          actions={actions}
          handleAction={this.handleAction}
          skipKustomize={this.skipKustomize}
          goBack={this.props.history.goBack}
          firstRoute={firstRoute}
        />
      );
    case "kustomize":
      return (
        <KustomizeOverlay
          startPoll={this.startPoll}
          getCurrentStep={this.getContentForStep}
          pollCallback={this.gotoRoute}
          routeId={routeId}
          actions={actions}
          isNavcycle={true}
          finalizeStep={this.props.finalizeStep}
          handleAction={this.handleAction}
          currentStep={currentStep}
          skipKustomize={this.skipKustomize}
          dataLoading={this.props.dataLoading}
          goBack={this.props.history.goBack}
          firstRoute={firstRoute}
        />
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
          <div className="flex-1-auto flex u-overflow--auto">
            {(isLoadingStep || dataLoading.getMetadataLoading) && !this.state.maxPollReached ?
              <div className="flex1 flex-column justifyContent--center alignItems--center">
                <Loader size="60" />
              </div>
              : this.state.maxPollReached ?
                <div className="flex1 flex-column justifyContent--center alignItems--center">
                  <p className="u-fontSize--large u-fontWeight--medium u-color--tundora">Oops, something isn't quite right. If you continue to experience this problem contact <a href="mailto:support@replicated.com">support@replicated.com</a></p>
                </div>
                :
                this.renderStep()
            }
          </div>
        </div>
      </div>
    );
  }
}

export default withRouter(DetermineComponentForRoute)
