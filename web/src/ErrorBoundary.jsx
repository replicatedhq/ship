import React from "react";

export default class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      error: null,
      errorInfo: null,
      hasError: false
    }
  }

  componentDidCatch(error, info) {
    this.setState({
      error: error,
      errorInfo: info,
      hasError: true
    });
    // We can thow an error modal, or some sort of error UI that can be used globally.
    // We should also send this error to bugsnag or some sort of reporting service here.
  }

  render() {
    return (
      <div className="flex-column flex1">
        {this.state.hasError ? <div>{this.state.error.toString()}</div> : null}
        {this.props.children}
      </div>
    );
  }
}