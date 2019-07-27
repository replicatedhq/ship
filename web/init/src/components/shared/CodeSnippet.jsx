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
    language: PropTypes.string,
    copyDelay: PropTypes.number
  }

  static defaultProps = {
    language: "bash",
    copyText: "Copy command",
    copyDelay: 3000
  }

  copySnippet = () => {
    const { children, copyDelay } = this.props;

    if (navigator.clipboard) {
      navigator.clipboard.writeText(children).then( () => {
        this.setState({ didCopy: true });

        setTimeout(() => {
          this.setState({ didCopy: false });
        }, copyDelay);
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
            <span className="CodeSnippet-copy u-fontWeight--bold" onClick={this.copySnippet}>
              {didCopy
                ? "Copied!"
                : copyText
              }
            </span>
          )}
        </div>
      </div>
    )
  }
}

export default CodeSnippet;
