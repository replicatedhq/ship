import React, { Component } from "react";
import PropTypes from "prop-types";

import CodeSnippet from "./CodeSnippet";

class PreflightChecksPage extends Component {
  constructor(props) {
    super(props);

    this.state = {
      showPreflightInstructions: false
    };
  }

  togglePreflightInstructions = () => {
    const { showPreflightInstructions } = this.state;
    this.setState({
      showPreflightInstructions: !showPreflightInstructions
    });
  }

  render() {
    const { showPreflightInstructions } = this.state;
    return (
      <div className="flex-column flex1">
        <div className="PreflightChecks--wrapper u-paddingTop--30">
          <div className="u-minWidth--full u-minHeight--full">
            <p className="u-fontSize--header u-color--tuna u-fontWeight--bold">
              Preflight checks
            </p>
            <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
              Preflight checks are designed to be run against a target cluster before installing an application. Preflights are simply a different set of collectors + analyzers. These checks are optional but are recommended to ensure that the application you install will work properly.
            </p>
            <p className="u-fontSize--jumbo u-color--tuna u-fontWeight--bold u-marginTop--30">
              Run this command from your workstation
            </p>
            <p className="u-marginTop--10 u-marginBottom-10">
              You will be able to see the results in your terminal window as well as in this UI.
            </p>
            <CodeSnippet className="u-marginTop--10" language="bash" canCopy={true}>
              kubectl preflight https://git.io/preflight-k8s.version.yaml
            </CodeSnippet>
            <div className="section-border flex justifyContent--center u-position--relative u-marginTop--20">
              <p
                className="preflight-button flex-auto u-fontSize--small u-color--astral u-fontWeight--medium u-cursor--pointer"
                onClick={this.togglePreflightInstructions}
              >
                {showPreflightInstructions
                  ? "I already have the preflight tool"
                  : "I need to install the preflight tool"
                }
              </p>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

export default PreflightChecksPage;
