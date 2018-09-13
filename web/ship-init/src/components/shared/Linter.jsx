import * as React from "react";
import autoBind from "react-autobind";
import Markdown from "react-remarkable";
import get from "lodash/get";

import "../../scss/components/shared/Linter.scss";

export default class Linter extends React.Component {

  constructor() {
    super();
    this.state = {
      showHelp: false,
      showPreview: true
    };
    autoBind(this);
  }

  maybeLineNumber(error) {
    return get(error, "positions.0.start.line")
      ? `Line ${error.positions[0].start.line + 1} | `
      : "";
  }

  maybeLevel(error) {
    return error.type === "error" ? "Error: " : error.type === "warn" ? "Warning: " : "";
  }

  messageFor(error) {
    return `${this.maybeLineNumber(error)}${this.maybeLevel(error)}${error.message}`
  }

  linkFor(error) {
    return error.links && error.links.length > 0 ? (
      <a target="_blank" href={error.links[0]} rel="noopener noreferrer">
        <span className={`icon u-linkIcon--blue flex-auto u-marginLeft--small clickable`}></span>
      </a>
    ) : null;
  }

  toggleWindow(window) {
    this.setState({
      showHelp: window === "help",
      showPreview: window === "preview"
    });
  }

  render() {
    const {
      errors,
      previewEnabled,
      readme
    } = this.props;
    const {
      showPreview,
      showHelp
    } = this.state;

    return (
      <div className="Linter-wrapper flex-column flex1 u-overflow--hidden">
        <div className="Linter-header flex flex-auto">
          <span className="flex1 u-fontSize--small u-fontWeight--medium u-color--dustyGray">Console</span>
        </div>
        <div className="Linter--console flex-1-auto flex-column u-overflow--auto">
          { previewEnabled &&
            <div className="Linter--toggle flex flex-auto justifyContent--center">
              <div className="flex">
                <span className={`${showHelp ? "is-active" : "not-active"}`} onClick={() => { this.toggleWindow("help") }}>Config Help</span>
                <span className={`${showPreview ? "is-active" : "not-active"}`} onClick={() => { this.toggleWindow("preview") }}>README.md</span>
              </div>
            </div>
          }
          { showPreview ?
            <div className="HelmChartReadme--wrapper markdown-wrapper readme flex flex1 flex-column u-overflow--auto">
              <Markdown
                options={{
                  linkTarget: "_blank",
                  linkify: true,
                }}>
                {readme}
              </Markdown>
            </div> :
            <div className="flex flex1 flex-column alignContent--center">
              {errors.length ?
                errors.map((error, i) => (
                  <div key={i} className={`ConsoleBlock ${error.type} flex alignSelf--flexStart`}>
                    <span className={`icon u-${error.type}ConsoleIcon flex-auto u-marginRight--small`}></span>
                    <span>{this.messageFor(error)}{this.linkFor(error)}</span>
                  </div>
                ))
                :
                <div className="flex1 flex-column alignItems--center justifyContent--center">
                  <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray">Everything looks good!</p>
                </div>
              }
            </div>
          }
        </div>
      </div>
    );
  }
}
