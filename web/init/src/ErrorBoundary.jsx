import React from "react";

/**
 * Creates an error boundary for any child component rendered inside
 */
export default class ErrorBoundary extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      error: null,
      errorInfo: null,
      hasError: false
    }
  }

  static getDerivedStateFromError(error) {
    return {
      error,
      hasError: true
    };
  }

  componentDidCatch() {
    // We can thow an error modal, or some sort of error UI that can be used globally.
    // We should also send this error to bugsnag or some sort of reporting service here.
  }

  render() {
    const { children } = this.props;
    const { hasError, error } = this.state;
    return (
      <div className="flex flex1">
        {hasError && <div>{error.toString()}</div>}
        {children}
      </div>
    );
  }
}