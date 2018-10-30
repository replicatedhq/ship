import React from "react";
import Modal from "react-modal";
import AceEditor from "react-ace";
import ReactTooltip from "react-tooltip"
import * as yaml from "js-yaml";
import isEmpty from "lodash/isEmpty";
import sortBy from "lodash/sortBy";
import pick from "lodash/pick";
import keyBy from "lodash/keyBy";
import find from "lodash/find";
import defaultTo from "lodash/defaultTo";
import debounce from "lodash/debounce";

import FileTree from "./FileTree";
import Loader from "../../shared/Loader";
import { AceEditorHOC, PATCH_TOKEN } from "./AceEditorHOC";
import DiffEditor from "../../shared/DiffEditor";

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
      viewDiff: false,
      markers: [],
      patch: "",
      savingFinalize: false,
      displayConfirmModal: false,
      overlayToDelete: "",
      addingNewResource: false,
      newResourceName: "",
      lastSavedPatch: null
    };
    this.addResourceWrapper = React.createRef();
    this.addResourceInput = React.createRef();
  }

  toggleModal = (overlayPath) => {
    this.setState({
      displayConfirmModal: !this.state.displayConfirmModal,
      overlayToDelete: this.state.displayConfirmModal ? "" : overlayPath
    });
  }

  componentDidUpdate(lastProps, lastState) {
    const { currentStep } = this.props;
    this.rebuildTooltip();
    if (this.props.currentStep !== lastProps.currentStep && !isEmpty(this.props.currentStep)) {
      this.setFileTree(currentStep);
    }
    if (this.props.fileContents !== lastProps.fileContents && !isEmpty(this.props.fileContents)) {
      this.setState({ fileContents: keyBy(this.props.fileContents, "key") });
    }
    if (
      (this.state.viewDiff !== lastState.viewDiff) ||
      (this.state.patch !== lastState.patch) ||
      (this.state.selectedFile !== lastState.selectedFile)
    ) {
      this.aceEditorOverlay.editor.resize();
    }
    if (this.props.patch !== lastProps.patch) {
      this.setState({
        lastSavedPatch: this.state.lastSavedPatch !== null ? this.state.lastSavedPatch : this.props.patch,
        patch: this.props.patch
      });
    }
  }

  componentDidMount() {
    const { currentStep } = this.props;
    if (currentStep && !isEmpty(currentStep)) {
      this.setFileTree(currentStep);
    }
    if (this.props.fileContents && !isEmpty(this.props.fileContents)) {
      this.setState({ fileContents: keyBy(this.props.fileContents, "key") });
    }
  }

  handleApplyPatch = async () => {
    const { selectedFile, fileTreeBasePath } = this.state;
    const contents = this.aceEditorOverlay.editor.getValue();

    const applyPayload = {
      resource: `${fileTreeBasePath}${selectedFile}`,
      patch: contents,
    };
    await this.props.applyPatch(applyPayload).catch();
  }

  toggleDiff = async () => {
    const { patch, modified } = this.props;
    const hasPatchButNoModified = patch.length > 0 && modified.length === 0;
    if (hasPatchButNoModified) {
      await this.handleApplyPatch().catch();
    }

    this.setState({ viewDiff: !this.state.viewDiff });
  }

  createOverlay = () => {
    const { selectedFile } = this.state;
    let file = find(this.props.fileContents, ["key", selectedFile]);
    if (!file) return;
    file = yaml.safeLoad(file.baseContent)
    const overlayFields = pick(file, "apiVersion", "kind", "metadata.name");
    const overlay = yaml.safeDump(overlayFields);
    this.setState({ patch: `--- \n${overlay}` });
  }

  setSelectedFile = async (path) => {
    const { lastSavedPatch, patch } = this.state;

    let canChangeFile = !lastSavedPatch || patch === lastSavedPatch || confirm("You have unsaved changes in the patch. If you proceed, you will lose any of the changes you've made.");
    if (canChangeFile) {
      this.setState({ selectedFile: path, lastSavedPatch: null });
      await this.props.getFileContent(path).then(() => {
        // set state with new file content
        this.setState({
          fileContents: keyBy(this.props.fileContents, "key"),
        });
      });
    }
  }

  handleFinalize = async () => {
    const {
      finalizeKustomizeOverlay,
      finalizeStep,
      history,
      isNavcycle,
      actions,
      startPoll,
      routeId,
      pollCallback
    } = this.props;

    if (isNavcycle) {
      await finalizeStep({ action: actions[0] });
      startPoll(routeId, pollCallback);
    } else {
      await finalizeKustomizeOverlay()
        .then(() => {
          this.setState({ savingFinalize: false });
          history.push("/");
        }).catch();
    }
  }

  discardOverlay = async () => {
    const { overlayToDelete } = this.state;
    await this.deleteOverlay(overlayToDelete);
    this.setState({
      patch: "",
      displayConfirmModal: false
    });
  }

  deleteOverlay = async (path) => {
    const { fileTree } = this.state;
    const resources = find(fileTree, { name: "resources" });
    const isResource = resources && !!find(resources.children, { path });
    await this.props.deleteOverlay(path, isResource);
  }

  handleKustomizeSave = async (finalize) => {
    const { selectedFile, fileContents } = this.state;
    const { isResource } = fileContents[selectedFile];
    const contents = this.aceEditorOverlay.editor.getValue();
    this.setState({ patch: contents });

    const payload = {
      path: selectedFile,
      contents,
      isResource
    };

    if (!isResource) await this.handleApplyPatch();
    await this.props.saveKustomizeOverlay(payload)
      .then(() => {
        this.setState({ lastSavedPatch: null });
      })
      .catch();
    await this.props.getCurrentStep();
    if (finalize) {
      this.setState({ savingFinalize: true, addingNewResource: false });
      this.handleFinalize();
    }
  }

  handleCreateResource = async () => {
    const { newResourceName } = this.state;
    const contents = "---"
    this.setState({ patch: contents });

    const payload = {
      path: `/${newResourceName}`,
      contents,
      isResource: true
    };

    await this.props.saveKustomizeOverlay(payload)
      .then(() => {
        this.setSelectedFile(`/${newResourceName}`);
        this.setState({ addingNewResource: false, newResourceName: "" })
      })
      .catch();
    await this.props.getCurrentStep();
  }

  handleGeneratePatch = async (path) => {
    const current = this.aceEditorOverlay.editor.getValue();
    const { selectedFile, fileTreeBasePath } = this.state;
    this.setState({ lastSavedPatch: null })
    const payload = {
      original: selectedFile,
      current,
      path,
      resource: `${fileTreeBasePath}${selectedFile}`,
    };
    await this.props.generatePatch(payload);
    this.aceEditorOverlay.editor.find(PATCH_TOKEN);
  }

  rebuildTooltip = () => {
    // We need to rebuild these because...well I dunno why but if you don't the tooltips will not be visible after toggling the overlay editor.
    ReactTooltip.rebuild();
    ReactTooltip.hide();
  }

  setFileTree = ({ kustomize }) => {
    if (!kustomize.tree) return;
    const sortedTree = sortBy(kustomize.tree.children, (dir) => {
      dir.children ? dir.children.length : 0
    });

    this.setState({
      fileTree: sortedTree,
      fileTreeBasePath: kustomize.basePath
    });
  }

  setAceEditor = (editor) => {
    this.aceEditorOverlay = editor;
  }

  updateModifiedPatch = debounce((patch, isResource) => {
    // We already circumvent React's lifecycle state system for updates
    // Set the current patch state to the changed value to avoid
    // React re-rendering the ACE Editor
    if (!isResource) {
      this.state.patch = patch; // eslint-disable-line
      this.handleApplyPatch();
    }
  }, 500);

  handleAddResourceClick = async () => {
    // Ref input won't focus until state has been set
    await this.setState({ addingNewResource: true });
    this.addResourceInput.current.focus();
    window.addEventListener("click", this.handleClickOutsideResourceInput);
  }

  handleClickOutsideResourceInput = (e) => {
    const { addingNewResource } = this.state;
    if (addingNewResource && !this.addResourceWrapper.current.contains(e.target)) {
      this.setState({ addingNewResource: false, newResourceName: "" });
      window.removeEventListener("click", this.handleClickOutsideResourceInput);
    }
  }

  handleCreateNewResource = (e) => {
    if (e.charCode === 13) {
      this.handleCreateResource()
    }
  }

  render() {
    const { dataLoading, modified } = this.props;
    const {
      fileTree,
      selectedFile,
      fileLoadErr,
      fileLoadErrMessage,
      patch,
      savingFinalize,
      fileContents,
      addingNewResource,
      newResourceName
    } = this.state;
    const fileToView = defaultTo(find(fileContents, ["key", selectedFile]), {});
    const showOverlay = patch.length;
    const showBase = !fileToView.isResource;

    return (
      <div className="flex flex1">
        <div className="u-minHeight--full u-minWidth--full flex-column flex1 u-position--relative">
          <div className="flex flex1 u-minHeight--full u-height--full">
            <div className="flex-column flex1 Sidebar-wrapper u-overflow--hidden">
              <div className="flex-column flex1">
                <div className="flex1 dirtree-wrapper flex-column u-overflow-hidden u-background--biscay">
                  {fileTree.map((tree, i) => (
                    <div className={`u-overflow--auto FileTree-wrapper u-position--relative dirtree ${i > 0 ? "flex-auto has-border" : "flex-0-auto"}`} key={i}>
                      <input type="checkbox" name={`sub-dir-${tree.name}-${tree.children.length}-${tree.path}-${i}`} id={`sub-dir-${tree.name}-${tree.children.length}-${tree.path}-${i}`} defaultChecked={true} />
                      <label htmlFor={`sub-dir-${tree.name}-${tree.children.length}-${tree.path}-${i}`}>{tree.name === "/" ? "base" : tree.name}</label>
                      < FileTree
                        files={tree.children}
                        basePath={tree.name}
                        handleFileSelect={(path) => this.setSelectedFile(path)}
                        handleDeleteOverlay={this.toggleModal}
                        selectedFile={this.state.selectedFile}
                        isOverlayTree={tree.name === "overlays"}
                        isResourceTree={tree.name === "resources"}
                      />
                    </div>
                  ))}
                  <div className="add-new-resource u-position--relative" ref={this.addResourceWrapper}>
                    <input
                      type="text"
                      className={`Input u-position--absolute ${!addingNewResource ? "u-visibility--hidden" : ""}`}
                      name="new-resource"
                      placeholder="filename.yaml"
                      onChange={(e) => { this.setState({ newResourceName: e.target.value }) }}
                      onKeyPress={(e) => { this.handleCreateNewResource(e) }}
                      value={newResourceName}
                      ref={this.addResourceInput}
                    />
                    <p
                      className={`add-resource-link u-position--absolute u-marginTop--small u-marginLeft--normal u-cursor--pointer u-fontSize--small u-color--silverSand u-fontWeight--bold ${addingNewResource ? "u-visibility--hidden" : ""}`}
                      onClick={this.handleAddResourceClick}
                    >+ Add Resource
                    </p>
                  </div>
                </div>
              </div>
            </div>
            <div className="flex-column flex1 u-height--auto u-overflow--hidden LayoutContent-wrapper u-position--relative">
              <div className="flex flex1 u-position--relative">

                <div className={`flex-column flex1 base-editor-wrapper ${showOverlay && "u-paddingRight--15"} ${showBase ? "visible" : ""}`}>
                  <div className="flex1 flex-column u-position--relative">
                    {fileLoadErr ?
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
                            <p className="u-fontSize--small u-lineHeight--more u-fontWeight--medium u-color--doveGray">Select a file to be used as the base YAML. You can then click the edit icon on the top right to create a patch for that file.</p>
                          </div>
                          {selectedFile !== "" ?
                            <div className="flex1 file-contents-wrapper AceEditor--wrapper">
                              {!showOverlay &&
                                <div data-tip="create-overlay-tooltip" data-for="create-overlay-tooltip" className="overlay-toggle u-cursor--pointer" onClick={this.createOverlay}>
                                  <span className="icon clickable u-overlayCreateIcon"></span>
                                </div>
                              }
                              <ReactTooltip id="create-overlay-tooltip" effect="solid" className="replicated-tooltip">Create patch</ReactTooltip>
                              <AceEditorHOC
                                handleGeneratePatch={this.handleGeneratePatch}
                                fileToView={fileToView}
                                diffOpen={this.state.viewDiff}
                                overlayOpen={showOverlay}
                              />
                            </div>
                            :
                            <div className="flex1 flex-column empty-file-wrapper alignItems--center justifyContent--center">
                              <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray">No file selected.</p>
                            </div>
                          }
                        </div>
                    }
                  </div>
                </div>

                <div className={`flex-column flex1 overlays-editor-wrapper ${showOverlay ? "visible" : ""}`}>
                  <div className="u-paddingLeft--20 u-paddingRight--20 u-paddingTop--20">
                    <p className="u-marginBottom--normal u-fontSize--large u-color--tuna u-fontWeight--bold">{showBase ? "Patch" : "Resource"}</p>
                    <p className="u-fontSize--small u-lineHeight--more u-fontWeight--medium u-color--doveGray">This YAML will be applied as a patch to the base YAML. Edit the values that you want patched. The current file you're editing will be automatically saved when you open a new file.</p>
                  </div>
                  <div className="flex1 flex-column file-contents-wrapper u-position--relative">
                    <div className="flex1 AceEditor--wrapper">
                      {showOverlay && showBase ? <span data-tip="close-overlay-tooltip" data-for="close-overlay-tooltip" className="icon clickable u-closeOverlayIcon" onClick={() => this.toggleModal(this.state.selectedFile)}></span> : null}
                      <ReactTooltip id="close-overlay-tooltip" effect="solid" className="replicated-tooltip">Discard patch</ReactTooltip>
                      <AceEditor
                        ref={this.setAceEditor}
                        mode="yaml"
                        theme="chrome"
                        className="flex1 flex"
                        value={patch || ""}
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
                        onChange={(patch) => this.updateModifiedPatch(patch, fileToView.isResource)}
                      />
                    </div>
                  </div>
                </div>
              </div>

              {showOverlay && showBase ?
                <div className={`${this.state.viewDiff ? "flex1" : "flex-auto"} flex-column`}>
                  <div className="diff-viewer-wrapper flex-column flex1">
                    <span className="diff-toggle" onClick={this.toggleDiff}>{this.state.viewDiff ? "Hide diff" : "Show diff"}</span>
                    {this.state.viewDiff &&
                      <DiffEditor
                        diffTitle="Diff YAML"
                        diffSubCopy="Here you can see the diff of the base YAML, and the finalized version with the overlay applied."
                        original={fileToView.baseContent}
                        updated={this.props.modified}
                      />
                    }
                  </div>
                </div>
                : null}

              <div className="flex-auto flex layout-footer-actions less-padding">
                <div className="flex1 flex-column flex-verticalCenter">
                  <p className="u-margin--none u-fontSize--small u-color--dustyGray u-fontWeight--normal">Contributed by <a target="_blank" rel="noopener noreferrer" href="https://replicated.com" className="u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover">Replicated</a></p>
                </div>
                <div className="flex1 flex alignItems--center justifyContent--flexEnd">
                  {selectedFile === "" ?
                    <button type="button" onClick={this.props.skipKustomize} className="btn primary">Continue</button>
                    :
                    <div className="flex">
                      <button type="button" disabled={dataLoading.saveKustomizeLoading || patch === "" || savingFinalize} onClick={() => this.handleKustomizeSave(false)} className="btn primary u-marginRight--normal">{dataLoading.saveKustomizeLoading && !savingFinalize ? "Saving patch" : "Save patch"}</button>
                      {patch === "" ?
                        <button type="button" onClick={this.props.skipKustomize} className="btn primary">Continue</button>
                        :
                        <button type="button" disabled={dataLoading.saveKustomizeLoading || savingFinalize} onClick={() => this.handleKustomizeSave(true)} className="btn secondary">{savingFinalize ? "Finalizing overlay" : "Save & continue"}</button>
                      }
                    </div>
                  }
                </div>
              </div>

            </div>
          </div>
        </div>
        <Modal
          isOpen={this.state.displayConfirmModal}
          onRequestClose={this.toggleModal}
          shouldReturnFocusAfterClose={false}
          ariaHideApp={false}
          contentLabel="Modal"
          className="Modal DefaultSize"
        >
          <div className="Modal-header">
            <p>Are you sure you want to discard this patch?</p>
          </div>
          <div className="flex flex-column Modal-body">
            <p className="u-fontSize--large u-fontWeight--normal u-color--dustyGray u-lineHeight--more">It will not be applied to the kustomization.yaml file that is generated for you.</p>
            <div className="flex justifyContent--flexEnd u-marginTop--20">
              <button className="btn secondary u-marginRight--10" onClick={() => this.toggleModal("")}>Cancel</button>
              <button type="button" className="btn primary" onClick={this.discardOverlay}>Discard patch</button>
            </div>
          </div>
        </Modal>
      </div>
    );
  }
}