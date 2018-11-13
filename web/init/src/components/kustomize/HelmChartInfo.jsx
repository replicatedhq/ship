import React from "react";
import Markdown from "react-remarkable";
import RenderActions from "../shared/RenderActions";

const HelmChartInfo = ({ shipAppMetadata, isUpdate, actions, handleAction, isLoading}) => {
  return (
    <div className="flex-column u-paddingTop--30 flex1 u-position--relative">
      <div className="flex-column flex-1-auto u-overflow--auto container">
        <div className="HelmIntro--wrapper flex-column">
          <p className="u-fontSize--jumbo2 u-color--tuna u-fontWeight--bold u-lineHeight--normal">Set custom values in your Helm chart</p>
          <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--more">
            {isUpdate ? `Let’s update the ${shipAppMetadata.name} application. Ship has downloaded the latest version of the Helm chart, and can merge your patches into the latest upstream, or allow you to edit your patches.` :
            `Let's prepare the ${shipAppMetadata.name} chart for a Production Grade deployment. To get started, we need to confirm the values to use when generating Kubernetes YAML from the Helm chart. Click continue to review the standard values provided in the chart.` }
          </p>
          <div className="HelmIntro--diagram flex">
            <div className="detailed-steps flex flex-column">
              <div className="detailed-step flex">
                <div className="icon helm-chart flex-auto"></div>
                <div className="flex flex-column">
                  <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Helm Chart</p>
                  <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--more u-fontWeight--medium">This is the original, upstream Helm Chart. You can use any Helm chart without forking here.</p>
                </div>
              </div>
              <div className="detailed-step flex">
                <div className="icon custom-values flex-auto"></div>
                <div className="flex flex-column">
                  <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Custom Values</p>
                  <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--more u-fontWeight--medium">Provide the values to the original Helm Chart. You will be able to view the generated YAML from these and have an opportunity to create patches using <a href="http://kustomize.io" target="_blank" rel="noopener noreferrer" className="u-color--astral u-fontWeight--medium">Kustomize</a> to make advanced changes without forking the original Helm Chart.</p>
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
            <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--more">
              {isUpdate ? "To get started, we need to confirm the values to use when generating the updated Kubernetes YAML from the latest version of the Helm chart. Click continue to review your values to provide to the chart." :
              "To get started, we need to confirm the values to use when generating Kubernetes YAML from the Helm chart. Click continue to review the standard values provided in the chart."}
            </p>
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
