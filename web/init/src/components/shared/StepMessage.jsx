import * as React from "react";
import Markdown from "react-remarkable";
import RenderActions from "./RenderActions";

const StepMessage = ({ actions, message, handleAction, isLoading }) => (
  <div className="StepMessage-wrapper flex flex1 flex-column">
    <div className={`markdown-wrapper flex flex1 flex-column u-overflow--auto  ${message.level || ""}`}>
      <Markdown
        options={{
          html: message.trusted_html,
          linkTarget: "_blank",
          linkify: true,
        }}>
        {message.contents}
      </Markdown>
    </div>
    <div className="u-marginTop--30 flex justifyContent--flexEnd">
      <RenderActions actions={actions} handleAction={handleAction} isLoading={isLoading} />
    </div>
  </div>
);

export default StepMessage;
