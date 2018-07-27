import * as React from "react";

const RenderActions = ({ actions, handleAction, isLoading }) => (
  <div>
    {!actions ? null : actions.map(
      (action, index) => {
        return (
          <button
            key={index}
            className={`btn ${action.buttonType}`}
            onClick={() => handleAction(action)}
            disabled={isLoading}>{isLoading ? action.loadingText : action.text}
          </button>
        );
      }
    )}
  </div>
);

export default RenderActions;
