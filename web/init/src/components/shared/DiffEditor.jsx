import React from "react";
import { MonacoDiffEditor } from "react-monaco-editor";

export default class DiffEditor extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      splitDiff: false
    };
  }

  toggleDiffType = () => {
    this.monacoDiffEditor.editor.updateOptions({
      renderSideBySide: !this.state.splitDiff
    });
    this.setState({ splitDiff: !this.state.splitDiff });
  }

  render() {
    const {
      original,
      updated,
      language,
      diffTitle,
      diffSubCopy,
      hideToggle,
    } = this.props;

    return (
      <div className="flex-column flex1">
        <div className="flex-auto flex">
          {diffTitle || diffSubCopy ?
            <div className="flex-auto u-marginBottom--normal">
              {diffTitle && <p className="u-fontSize--large u-color--tuna u-fontWeight--bold">{diffTitle}</p>}
              {diffSubCopy && <p className="u-marginTop--small u-fontSize--small u-lineHeight--more u-fontWeight--medium u-color--doveGray">{diffSubCopy}</p>}
            </div>
            : null}
          {hideToggle ? null :
            <div className="flex flex1 justifyContent--flexEnd">
              <div className="flex-column flex-auto flex-verticalCenter">
                <div className="diff-type-toggle flex flex-auto">
                  <span className={`${!this.state.splitDiff ? "is-active" : "not-active"} flex-auto`} onClick={this.toggleDiffType}>Inline diff</span>
                  <span className={`${this.state.splitDiff ? "is-active" : "not-active"} flex-auto`} onClick={this.toggleDiffType}>Split diff</span>
                </div>
              </div>
            </div>
          }
        </div>
        <div className="flex1">
          <MonacoDiffEditor
            ref={(editor) => { this.monacoDiffEditor = editor}}
            width="100%"
            height="100%"
            language={language || "yaml"}
            original={original}
            value={updated}
            options={{
              renderSideBySide: false,
              enableSplitViewResizing: true,
              scrollBeyondLastLine: false,
            }}
          />
        </div>
      </div>
    );
  }
}