import React from "react";
import Markdown from "react-remarkable";
import RenderActions from "../shared/RenderActions";

const HelmChartInfo = ({ shipAppMetadata, actions, handleAction, isLoading}) => {
  const { readme, version } = shipAppMetadata;
  return (
    <div className="flex-column u-paddingTop--30 flex1 u-position--relative">
      <div className="flex-column flex-1-auto u-overflow--auto container">
        <div className="HelmIntro--wrapper flex-column">
          <p className="u-fontSize--jumbo2 u-color--tuna u-fontWeight--bold u-lineHeight--normal">Set custom values in youlr Helm chart</p>
          <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--more">Let’s prepare the CoreOS vault-operator chart for a Production Grade deployment. To get started, we need to confirm the values to use when generating Kubernetes YAML from the Helm chart. Click continue to review the standard values provided in the chart.</p>
          <div className="HelmIntro--diagram flex">
            <div className="detailed-steps flex flex-column">
              <div className="detailed-step flex">
                <div className="icon helm-chart flex-auto"></div>
                <div className="flex flex-column">
                  <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Helm Chart</p>
                  <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--more u-fontWeight--medium">The base is the rendered Helm chart. Ship will create this for you and it’s generated from the original Chart, never forked.</p>
                </div>
              </div>
              <div className="detailed-step flex">
                <div className="icon custom-values flex-auto"></div>
                <div className="flex flex-column">
                  <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Custom Values</p>
                  <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--more u-fontWeight--medium">The changes you would have made in a fork, i.e. any advanced customization (additions, deletions or changes) to the base can be written as patches and will be managed using <a href="http://kustomize.io" target="_blank" rel="noopener noreferrer" className="u-color--astral u-fontWeight--medium">Kustomize</a>. Ship will guide you through creating these patches.</p>
                </div>
              </div>
            </div>
            <div className="border-wth-arrow flex flex-column alignItems--center">
              <div className="line flex1"></div>
              <div className="icon arrow flex-auto"></div>
              <div className="line flex1"></div>
            </div>
            <div className="kustomize-steps flex flex-auto alignItems--center">
              <div className="flex flex-column alignItems--center">
                <div className="icon base-small"></div>
                <p className="u-fontSize--small u-color--tuna u-fontWeight--bold u-marginTop--normal u-lineHeight--small">Base</p>
              </div>
              <p className="plus u-color--chateauGreen u-fontSize--jumbo2 u-fontWeight--bold">+</p>
              <div className="flex flex-column alignItems--center">
                <div className="icon patches-small"></div>
                <p className="u-fontSize--small u-color--tuna u-fontWeight--bold u-marginTop--normal u-lineHeight--small">Patches</p>
              </div>
            </div>
            <div className="flex alignItems--center u-marginRight--small u-marginLeft--small">
              <p className="plus u-color--chateauGreen u-fontSize--jumbo2 u-fontWeight--bold">=</p>
            </div>
            <div className="deployment-step flex alignItems--center">
              <div className="flex flex-column alignItems--center">
                <div className="icon deployable-app"></div>
                <p className="u-textAlign--center u-fontSize--small u-color--tuna u-fontWeight--bold u-marginTop--normal u-lineHeight--small">Deployable App</p>
              </div>
            </div>
          </div>
          <div className="flex flex-column flex1 u-borderTop--gray">
            <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--more">To get started, we need to confirm the values to use when generating Kubernetes YAML from the Helm chart. Click continue to review the standard values provided in the chart.</p>
            <p className="u-marginTop--20 u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--more">Happy Shipping!</p>
          </div>
        </div>
      </div>
      <div className="actions-wrapper container u-width--full flex flex-auto justifyContent--flexEnd">
        <RenderActions actions={actions} handleAction={handleAction} isLoading={isLoading} />
      </div>
    </div>
  )
}

export default HelmChartInfo;
