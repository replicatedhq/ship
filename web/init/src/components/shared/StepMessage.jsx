import * as React from "react";
import Markdown from "react-remarkable";
import RenderActions from "./RenderActions";

const StepMessage = ({ actions, message, handleAction, goBack, firstRoute, isLoading }) => (
  <div className="StepMessage-wrapper flex1 flex-column">
    <div className={`markdown-wrapper flex1 flex-column u-overflow--auto  ${message.level || ""}`}>
      <div className="mkdwn">
        <Markdown
          options={{
            html: message.trusted_html,
            linkTarget: "_blank",
            linkify: true,
          }}>
          {message.contents}
        </Markdown>
      </div>
    </div>
    <div className="flex flex-auto actions-wrapper u-paddingRight--20 u-paddingLeft--20">
      {firstRoute ? null :
        <div className="flex-auto u-marginRight--normal">
          <button className="btn secondary" onClick={() => goBack()}>Back</button>
        </div>
      }
      <div className="flex1 flex justifyContent--flexEnd">
        <RenderActions actions={actions} handleAction={handleAction} isLoading={isLoading} />
      </div>
    </div>
  </div>
);

export default StepMessage;
