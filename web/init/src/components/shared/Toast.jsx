import * as React from "react";
import autoBind from "react-autobind";

import "../../scss/components/shared/Toast.scss";

export default class Toast extends React.Component {
  constructor() {
    super();
    autoBind(this);
  }

  render() {
    const { toast, onCancel } = this.props;

    return (
      <div className={`Toast-wrapper ${toast.showToast ? "visible": ""} ${toast.type} flex flex1`}>
        <div className="flex1 flex-column flex-verticalCenter">
          <div className="flex-auto">
            <div className="flex">
              {toast.opts.showCancelButton && <div className="flex-column flex-verticalCenter"><span onClick={onCancel} className="icon clickable u-closeIcon u-marginRight--normal">{toast.opts.cancelButtonText}</span></div>}
              <p className="Toast-title">{toast.title}</p>
            </div>
            {toast.subText && <div className="Toast-sub">{toast.subText}</div>}
          </div>
        </div>
        <div className="flex-auto flex-column flex-verticalCenter">
          <div className="flex">
            <button onClick={toast.opts.confirmAction} className="btn primary">{toast.opts.confirmButtonText}</button>
          </div>
        </div>
      </div>
    );
  }
}
