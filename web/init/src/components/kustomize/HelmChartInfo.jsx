import React from "react";
import Markdown from "react-remarkable";
import RenderActions from "../shared/RenderActions";

const HelmChartInfo = ({ shipAppMetadata, actions, handleAction, isLoading}) => {
  const { readme, version } = shipAppMetadata;
  return (
    <div className="flex-column u-paddingTop--30 flex1 u-position--relative">
      <div className="flex-column flex-1-auto u-overflow--auto container">
        <div className="HelmChartInfo--wrapper flex-column">
          <div className="flex flex1 u-marginBottom--normal u-paddingLeft--15">
            <p className="u-color--dustyGray u-fontWeight--medium u-fontSize--normal">Version <span className="u-fontWeight--bold u-color--tuna">{ version }</span></p>
          </div>
          <div className="HelmChartReadme--wrapper markdown-wrapper">
            <Markdown
              options={{
                linkTarget: "_blank",
                linkify: true,
              }}>
              { readme }
            </Markdown>
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
