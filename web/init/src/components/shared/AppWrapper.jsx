import React from "react";
import DocumentTitle from "react-document-title";

export default class AppWrapper extends React.Component {
  render() {
    const { channelDetails } = this.props;
    return (
      <DocumentTitle title={`${channelDetails.channelName && channelDetails.channelName.length ? channelDetails.channelName : "Enterprise"} Deployment Generator | Ship`}>
        <div className="u-minHeight--full u-minWidth--full flex-column flex1">
          {this.props.children}
        </div>
      </DocumentTitle>
    );
  }
}
