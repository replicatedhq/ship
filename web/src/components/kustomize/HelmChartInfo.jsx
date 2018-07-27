import React from "react";
import Markdown from "react-remarkable";
import RenderActions from "../shared/RenderActions";

import "../../scss/components/kustomize/HelmChartInfo.scss";
import "../../scss/components/shared/DetermineStep.scss";

const HelmChartInfo = ({ helmChartMetadata, actions, handleAction, isLoading}) => {
  const { readme, version } = helmChartMetadata;
  return (
    <div className="HelmChartInfo--wrapper flex-column flex1 u-position--relative">
      <div className="flex flex1 u-marginBottom--normal u-paddingLeft--15">
        <p className="u-color--dustyGray u-fontWeight--medium u-fontSize--normal">Version <span className="u-fontWeight--bold u-color--tuna">{ version }</span></p>
      </div>
      <div className="HelmChartReadme--wrapper markdown-wrapper">
        <Markdown>{ readme }</Markdown>
      </div>
      <div className="action container u-width--full u-marginTop--30 flex flex1 justifyContent--flexEnd u-position--fixed u-bottom--0 u-right--0 u-left--0">
        <RenderActions actions={actions} handleAction={handleAction} isLoading={isLoading} />
      </div> 
    </div>
  )
}

export default HelmChartInfo;
