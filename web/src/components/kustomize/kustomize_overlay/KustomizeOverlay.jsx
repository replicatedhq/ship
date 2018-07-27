import React from "react";
import autoBind from "react-autobind";
import AceEditor from "react-ace";
import ReactTooltip from "react-tooltip"
import * as yaml from "js-yaml";
import isEmpty from "lodash/isEmpty";
import sortBy from "lodash/sortBy";
import pick from "lodash/pick";
import keyBy from "lodash/keyBy";
import find from "lodash/find";

import FileTree from "./FileTree";
import Loader from "../../shared/Loader";
import Toast from "../../shared/Toast";
import KustomizeEmpty from "./KustomizeEmpty";

import "../../../scss/components/kustomize/KustomizeOverlay.scss";
import "../../../../node_modules/brace/mode/yaml";
import "../../../../node_modules/brace/theme/chrome";

export default class KustomizeOverlay extends React.Component {
  constructor() {
    super();
    this.state = {
      fileTree: [],
      fileTreeBasePath: "",
      selectedFile: "",
      fileContents: {},
      fileLoadErr: false,
      fileLoadErrMessage: "",
      addOverlay: false,
      overlayContent: "",
      toastDetails: {
        opts: {}
      }
    };
    autoBind(this);
  }

  openOverlay() {
    this.setState({ addOverlay: true });
  }

  discardOverlay() {
    const file = find(this.props.fileContents, ["key", this.state.selectedFile]);
    const initalOverlay = file.overlayContent;
    this.setState({ addOverlay: false, overlayContent: initalOverlay });
  }

  createOverlay() {
    const { fileContents, selectedFile } = this.state;
    let file = fileContents[selectedFile];
    if (!file) return;
    file = yaml.safeLoad(file.baseContent)
    const overlayFields = pick(file, "apiVersion", "kind", "metadata.name");
    const overlay = yaml.safeDump(overlayFields);
    this.setState({ overlayContent: `--- \n${overlay}` });
    this.openOverlay();
  }

  hasContentAlready(path) {
    const { fileContents } = this.state;
    if (fileContents[path]) { return true; }
    return false;
  }

  async setSelectedFile(path) {
    this.setState({ selectedFile: path });
    if (this.state.toastDetails.showToast) {
      this.cancelToast();
    }
    if (this.hasContentAlready(path)) {
      // if we've already fetched the file, set the overlayContent from existing content
      const file = this.state.fileContents[path];
      this.setState({ overlayContent: file.overlayContent });
      return;
    }
    await this.props.getFileContent(path).then(() => {
      // set state with new file content and set the overlayContent from new file content on the file the user wants to view
      const file = find(this.props.fileContents, ["key", path]);
      this.setState({
        fileContents: keyBy(this.props.fileContents, "key"),
        overlayContent: file.overlayContent
      });
    });
  }

  cancelToast() {
    let nextState = {};
    nextState.toastDetails = {
      showToast: false,
      title: "",
      subText: "",
      type: "",
      opts: {}
    };
    this.setState(nextState)
  }

  async handlFinalize() {
    await this.props.finalizeKustomizeOverlay()
      .then(() => {
        this.props.history.push("/");
      }).catch();
  }

  onKustomizeSaved() {
    let nextState = {};
    nextState.toastDetails = {
      showToast: true,
      title: "Overlay has been saved.",
      type: "success",
      opts: {
        showCancelButton: true,
        confirmButtonText: "Finalize overlays",
        confirmAction: () => this.handlFinalize()
      }
    }
    this.setState(nextState);
  }

  async handleKustomizeSave() {
    const { selectedFile, overlayContent } = this.state;
    const payload = {
      path: selectedFile,
      contents: overlayContent
    }
    await this.props.saveKustomizeOverlay(payload).then(() => {
      this.onKustomizeSaved();
    }).catch();
  }

  rebuildTooltip() {
    // We need to rebuild these because...well I dunno why but if you don't the tooltips will not be visible after toggling the overlay editor.
    ReactTooltip.rebuild();
    ReactTooltip.hide();
  }

  setFileTree() {
    const { kustomize } = this.props.currentStep;
    if (!kustomize.tree) return;
    let sortedTree = sortBy([kustomize.tree], (dir) => {
      dir.children ? dir.children.length : []
    });
    sortedTree.reverse();
    const basePath = kustomize.basePath.substr(kustomize.basePath.lastIndexOf("/") + 1);
    this.setState({
      fileTree: sortedTree,
      fileTreeBasePath: basePath
    });
  }

  componentDidUpdate(lastProps, lastState) {
    this.rebuildTooltip();
    if (this.props.currentStep !== lastProps.currentStep && !isEmpty(this.props.currentStep)) {
      this.setFileTree();
    }
    if (this.props.fileContents !==lastProps.fileContents && !isEmpty(this.props.fileContents)) {
      this.setState({ fileContents: keyBy(this.props.fileContents, "key") });
    }
    if (this.state.addOverlay !== lastState.addOverlay && this.state.addOverlay) {
      if (this.refs.aceEditorOverlay) {
        this.refs.aceEditorOverlay.editor.resize();
      }
    }
  }

  componentDidMount() {
    if (isEmpty(this.props.currentStep)) {
      this.props.getCurrentStep()
    }
    if (this.props.currentStep && !isEmpty(this.props.currentStep)) {
      this.setFileTree();
    }
    if (this.props.fileContents && !isEmpty(this.props.fileContents)) {
      this.setState({ fileContents: keyBy(this.props.fileContents, "key") });
    }
  }

  render() {
    const { dataLoading } = this.props;
    const { fileTree, fileTreeBasePath, selectedFile, fileContents, fileLoadErr, fileLoadErrMessage, overlayContent, toastDetails } = this.state;
    const fileToView = fileContents[selectedFile];

    return (
      <div className="flex flex1">
        <div className="u-minHeight--full u-minWidth--full flex-column flex1 u-position--relative">
          <div className="flex flex1 u-minHeight--full u-height--full">
            <div className="flex-column flex1 Sidebar-wrapper u-overflow--hidden">
              <div className="flex-column flex1">
                <div className={`flex1 dirtree-wrapper flex-column u-overflow-hidden u-background--biscay ${this.props.isFullscreen ? "fs-mode" : ""}`}>
                  <div className="u-overflow--auto dirtree">
                    <FileTree
                      files={fileTree}
                      basePath={fileTreeBasePath}
                      isRoot={true}
                      handleFileSelect={(path) => this.setSelectedFile(path)}
                      selectedFile={this.state.selectedFile}
                    />
                  </div>
                </div>
              </div>
            </div>
            <div className="flex-column flex1 u-height--auto u-overflow--hidden LayoutContent-wrapper u-position--relative">
              <Toast toast={toastDetails} onCancel={this.cancelToast} />
              <div className="flex flex1 u-position--relative">

                <div className={`flex-column flex1 ${this.state.addOverlay && "u-paddingRight--15"}`}>
                  <div className="flex1 flex-column u-position--relative">
                    {selectedFile === "" ?
                      <KustomizeEmpty skipKustomize={() => this.handlFinalize()} />
                      : fileLoadErr ?
                        <div className="flex-column flex1 alignItems--center justifyContent--center">
                          <p className="u-color--chestnut u-fontSize--normal u-fontWeight--medium">Oops, we ran into a probelm getting that file, <span className="u-fontWeight--bold">{fileLoadErrMessage}</span></p>
                        </div>
                        : dataLoading.fileContentLoading ?
                          <div className="flex-column flex1 alignItems--center justifyContent--center">
                            <Loader size="50" color="#337AB7" />
                          </div>
                          :
                          <div className="flex1 flex-column">
                            <div className="u-paddingLeft--20 u-paddingRight--20 u-paddingTop--20">
                              <p className="u-marginBottom--normal u-fontSize--large u-color--tuna u-fontWeight--bold">Base YAML</p>
                              <p className="u-fontSize--small u-lineHeight--more u-fontWeight--medium u-color--doveGray">Select a file to be used as the base YAML. You can then click the edit icon on the top right to create an overlay for that file.</p>
                            </div>
                            <div className="flex1 file-contents-wrapper AceEditor--wrapper">
                              {!this.state.addOverlay &&
                              <div data-tip="create-overlay-tooltip" data-for="create-overlay-tooltip" className="overlay-toggle u-cursor--pointer" onClick={this.createOverlay}>
                                <span className="icon clickable u-overlayCreateIcon"></span>
                              </div>
                              }
                              <ReactTooltip id="create-overlay-tooltip" effect="solid" className="replicated-tooltip">Create overlay</ReactTooltip>
                              <AceEditor
                                ref="aceEditorBase"
                                mode="yaml"
                                theme="chrome"
                                className="flex1 flex disabled-ace-editor ace-chrome"
                                readOnly={true}
                                value={fileToView && fileToView.baseContent || ""}
                                height="100%"
                                width="100%"
                                editorProps={{
                                  $blockScrolling: Infinity,
                                  useSoftTabs: true,
                                  tabSize: 2,
                                }}
                                setOptions={{
                                  scrollPastEnd: false
                                }}
                              />
                            </div>
                          </div>
                    }
                  </div>
                </div>

                <div className={`flex-column flex1 overlays-editor-wrapper ${this.state.addOverlay || (fileToView && fileToView.overlayContent.length) ? "visible" : ""}`}>
                  <div className="u-paddingLeft--20 u-paddingRight--20 u-paddingTop--20">
                    <p className="u-marginBottom--normal u-fontSize--large u-color--tuna u-fontWeight--bold">Overlay</p>
                    <p className="u-fontSize--small u-lineHeight--more u-fontWeight--medium u-color--doveGray">This YAML will be applied as an overlay to the base YAML. Edit the values that you want overlayed. The current file you're editing will be automatically save when you open a new file.</p>
                  </div>
                  <div className="flex1 flex-column file-contents-wrapper u-position--relative">
                    <div className="flex1 AceEditor--wrapper">
                      {this.state.addOverlay && <span data-tip="discard-overlay-tooltip" data-for="discard-overlay-tooltip" className="icon clickable u-discardOverlayIcon" onClick={this.discardOverlay}></span>}
                      <ReactTooltip id="discard-overlay-tooltip" effect="solid" className="replicated-tooltip">Discard overlay</ReactTooltip>
                      <AceEditor
                        ref="aceEditorOverlay"
                        mode="yaml"
                        theme="chrome"
                        className="flex1 flex"
                        value={overlayContent || ""}
                        height="100%"
                        width="100%"
                        editorProps={{
                          $blockScrolling: Infinity,
                          useSoftTabs: true,
                          tabSize: 2,
                        }}
                        setOptions={{
                          scrollPastEnd: false
                        }}
                      />
                    </div>
                  </div>
                </div>
              </div>

              <div className="flex-auto flex layout-footer-actions less-padding">
                <div className="flex1 flex-column flex-verticalCenter">
                  <p className="u-margin--none u-fontSize--small u-color--dustyGray u-fontWeight--normal">Contributed by <a target="_blank" rel="noopener noreferrer" href="https://replicated.com" className="u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover">Replicated</a></p>
                </div>
                <div className="flex1 flex justifyContent--flexEnd">
                  <button type="button" disabled={dataLoading.saveKustomizeLoading} onClick={this.handleKustomizeSave} className="btn primary">{dataLoading.saveKustomizeLoading ? "Saving overlays"  : "Save overlays"}</button>
                </div>
              </div>

            </div>
          </div>
        </div>
      </div>
    );
  }
}