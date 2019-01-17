import * as React from "react";

export default class Toast extends React.Component {
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
      </div>
    );
  }
}
