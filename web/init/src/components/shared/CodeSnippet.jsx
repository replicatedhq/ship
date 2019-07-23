import React, { Component } from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import Prism from "@maji/react-prism";

class CodeSnippet extends Component {
  state = {
    didCopy: false
  }
  static propTypes = {
    children: PropTypes.string.isRequired,
    canCopy: PropTypes.bool,
    language: PropTypes.string
  }

  static defaultProps = {
    language: "bash",
    copyText: "Copy command"
  }

  copySnippet = () => {
    const { children } = this.props;

    if (navigator.clipboard) {
      navigator.clipboard.writeText(children).then( () => {
        this.setState({ didCopy: true });

        setTimeout(() => {
          this.setState({ didCopy: false });
        }, 3000);
      });
    }
  }

  render() {
    const {
      className,
      children,
      language,
      canCopy,
      copyText
    } = this.props;

    const { didCopy } = this.state;

    return (
      <div className={classNames("CodeSnippet", className)}>
        <div className="CodeSnippet-content">
          <Prism language={language}>
            {children}
          </Prism>
          {canCopy && (
            <p className="CodeSnippet-copy u-fontWeight--bold" onClick={this.copySnippet}>
              {didCopy
                ? "Copied!"
                : copyText
              }
            </p>
          )}
        </div>
      </div>
    )
  }
}

export default CodeSnippet;
