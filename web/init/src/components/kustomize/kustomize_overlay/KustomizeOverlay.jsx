import React from "react";
import AceEditor from "react-ace";
import ReactTooltip from "react-tooltip"
import * as yaml from "js-yaml";
import isEmpty from "lodash/isEmpty";
import sortBy from "lodash/sortBy";
import pick from "lodash/pick";
import keyBy from "lodash/keyBy";
import find from "lodash/find";
import trim from "lodash/trim";
import findIndex from "lodash/findIndex";
import map from "lodash/map";
import defaultTo from "lodash/defaultTo";

import FileTree from "./FileTree";
import KustomizeModal from "./KustomizeModal";
import Loader from "../../shared/Loader";
import { AceEditorHOC, PATCH_TOKEN } from "./AceEditorHOC";
import DiffEditor from "../../shared/DiffEditor";

import "../../../../node_modules/brace/mode/yaml";
import "../../../../node_modules/brace/theme/chrome";

export const PATCH_OVERLAY = "PATCH";
export const BASE_OVERLAY = "BASE";
export const RESOURCE_OVERLAY = "RESOURCE";

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
      savePatchErr: false,
      savePatchErrorMessage: "",
      applyPatchErr: false,
      applyPatchErrorMessage: "",
      viewDiff: false,
      markers: [],
      patch: "",
      savingFinalize: false,
      displayConfirmModal: false,
      overlayToDelete: "",
      addingNewResource: false,
      newResourceName: "",
      lastSavedPatch: null,
      displayConfirmModalMessage: "",
      displayConfirmModalDiscardMessage: "",
      displayConfirmModalSubMessage: "",
      modalAction: this.discardOverlay,
    };
    this.addResourceWrapper = React.createRef();
    this.addResourceInput = React.createRef();
  }

  toggleModal = (overlayPath, overlayType) => {
    const displayConfirmModalSubMessage = "It will not be applied to the kustomization.yaml file that is generated for you.";
    let displayConfirmModalMessage = "Are you sure you want to discard this patch?";
    let displayConfirmModalDiscardMessage = "Discard patch";

    if (overlayType === BASE_OVERLAY) {
      displayConfirmModalMessage = "Are you sure you want to discard this base resource?";
      displayConfirmModalDiscardMessage = "Discard base";
    } else if (overlayType === RESOURCE_OVERLAY) {
      displayConfirmModalMessage = "Are you sure you want to discard this resource?";
      displayConfirmModalDiscardMessage = "Discard resource";
    }

    this.setState({
      displayConfirmModal: !this.state.displayConfirmModal,
      overlayToDelete: this.state.displayConfirmModal ? "" : overlayPath,
      displayConfirmModalMessage,
      displayConfirmModalDiscardMessage,
      displayConfirmModalSubMessage,
      modalAction: () => (this.discardOverlay(overlayType)),
    });
  }

  toggleModalForExcludedBase = (basePath) => {
    this.setState({
      displayConfirmModal: !this.state.displayConfirmModal,
      displayConfirmModalMessage: "Are you sure you want to include this base resource?",
      displayConfirmModalDiscardMessage: "Include base",
      displayConfirmModalSubMessage: "It will be included in the kustomization.yaml file that is generated for you.",
      modalAction: () => (this.includeBase(basePath)),
    });
  }

  includeBase = async(basePath) => {
    await this.props.includeBase(basePath);
    this.setState({ displayConfirmModal: false });
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
    await this.props.applyPatch(applyPayload)
      .catch((err) => {
        this.setState({ 
          applyPatchErr: true,
          applyPatchErrorMessage: err.message 
        });

        setTimeout(() => {
          this.setState({ 
            applyPatchErr: false,
            applyPatchErrorMessage: "" 
          });
        }, 3000);
      });
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
    const files = yaml.safeLoadAll(file.baseContent);
    let overlayFields = map(files, (file) => {
      return pick(file, "apiVersion", "kind", "metadata.name")
    });
    if (files.length === 1) {
      overlayFields = overlayFields[0];
    }
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

  discardOverlay = async (overlayType) => {
    const { overlayToDelete } = this.state;
    await this.deleteOverlay(overlayToDelete, overlayType);
    this.setState({
      patch: "",
      displayConfirmModal: false,
      lastSavedPatch: null
    });
  }

  deleteOverlay = async (path, overlayType) => {
    const { fileTree, selectedFile } = this.state;
    const isResource = overlayType === RESOURCE_OVERLAY;
    const isBase = overlayType === BASE_OVERLAY;

    const overlays = find(fileTree, { name: "overlays" });
    const overlayExists = overlays && findIndex(overlays.children, { path }) > -1;

    if (isResource) {
      await this.props.deleteOverlay(path, "resource");
      return;
    }

    if (isBase) {
      if (selectedFile === path) {
        this.setState({ selectedFile: "" });
      }
      await this.props.deleteOverlay(path, "base");
      return;
    }

    if (overlayExists) {
      await this.props.deleteOverlay(path, "patch");
      return;
    }
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
      .catch((err) => {
        this.setState({
          savePatchErr: true,
          savePatchErrorMessage: err.message
        });

        setTimeout(() => {
          this.setState({
            savePatchErr: false,
            savePatchErrorMessage: ""
          });
        }, 3000);
      });
    await this.props.getCurrentStep();
    if (finalize) {
      this.setState({ savingFinalize: true, addingNewResource: false });
      this.handleFinalize();
    }
  }

  handleCreateResource = async () => {
    const { newResourceName } = this.state;
    const contents = "\n"; // Cannot be empty
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
      .catch((err) => {
        this.setState({
          savePatchErr: true,
          savePatchErrorMessage: err.message
        });

        setTimeout(() => {
          this.setState({
            savePatchErr: false,
            savePatchErrorMessage: ""
          });
        }, 3000);
      });
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

    const position = this.aceEditorOverlay.editor.find(PATCH_TOKEN); // Find text for position
    if(position) {
      this.aceEditorOverlay.editor.focus();
      this.aceEditorOverlay.editor.gotoLine(position.start.row + 1, position.start.column);
      this.aceEditorOverlay.editor.find(PATCH_TOKEN); // Have to find text again to auto focus text
    }
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

  updateModifiedPatch = (patch, isResource) => {
    // We already circumvent React's lifecycle state system for updates
    // Set the current patch state to the changed value to avoid
    // React re-rendering the ACE Editor
    if (!isResource) {
      this.state.patch = patch; // eslint-disable-line
    }
  };

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
    const { dataLoading, modified, firstRoute, goBack } = this.props;
    const {
      fileTree,
      selectedFile,
      fileLoadErr,
      fileLoadErrMessage,
      patch,
      savingFinalize,
      fileContents,
      addingNewResource,
      newResourceName,
      modalAction,
      applyPatchErr,
      applyPatchErrorMessage,
      savePatchErr,
      savePatchErrorMessage
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
                      <FileTree
                        files={tree.children}
                        basePath={tree.name}
                        handleFileSelect={(path) => this.setSelectedFile(path)}
                        handleDeleteOverlay={this.toggleModal}
                        handleClickExcludedBase={this.toggleModalForExcludedBase}
                        selectedFile={this.state.selectedFile}
                        isOverlayTree={tree.name === "overlays"}
                        isResourceTree={tree.name === "resources"}
                        isBaseTree={tree.name === "/"}
                      />
                    </div>
                  ))}
                  <div className="add-new-resource u-position--relative" ref={this.addResourceWrapper}>
                    <input
                      type="text"
                      className={`Input add-resource-name-input u-position--absolute ${!addingNewResource ? "u-visibility--hidden" : ""}`}
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
                                handleApplyPatch={this.handleApplyPatch}
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
                    <p className="u-fontSize--small u-lineHeight--more u-fontWeight--medium u-color--doveGray">This file will be applied as a patch to the base manifest. Edit the values that you want patched. The current file you're editing will be automatically saved when you open a new file.</p>
                  </div>
                  <div className="flex1 flex-column file-contents-wrapper u-position--relative">
                    <div className="flex1 AceEditor--wrapper">
                      {showOverlay && showBase ? <span data-tip="close-overlay-tooltip" data-for="close-overlay-tooltip" className="icon clickable u-closeOverlayIcon" onClick={() => this.toggleModal(this.state.selectedFile, PATCH_OVERLAY)}></span> : null}
                      <ReactTooltip id="close-overlay-tooltip" effect="solid" className="replicated-tooltip">Discard patch</ReactTooltip>
                      <AceEditor
                        ref={this.setAceEditor}
                        mode="yaml"
                        theme="chrome"
                        className="flex1 flex acePatchEditor"
                        value={trim(patch)}
                        height="100%"
                        width="100%"
                        editorProps={{
                          $blockScrolling: Infinity,
                          useSoftTabs: true,
                          tabSize: 2,
                        }}
                        debounceChangePeriod={1000}
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
                <div className="flex flex1">
                  {firstRoute ? null :
                    <div className="flex-auto u-marginRight--normal">
                      <button className="btn secondary" onClick={() => goBack()}>Back</button>
                    </div>
                  }
                  <div className="flex-column flex-verticalCenter">
                    <p className="u-margin--none u-marginRight--30 u-fontSize--small u-color--dustyGray u-fontWeight--normal">Contributed by <a target="_blank" rel="noopener noreferrer" href="https://replicated.com" className="u-fontWeight--medium u-color--astral u-textDecoration--underlineOnHover">Replicated</a></p>
                  </div>
                </div>
                <div className="flex1 flex alignItems--center justifyContent--flexEnd">
                  {selectedFile === "" ?
                    <button type="button" onClick={this.props.skipKustomize} className="btn primary">Continue</button>
                    :
                    <div className="flex">
                      {applyPatchErr && <span className="flex flex1 u-fontSize--small u-fontWeight--medium u-color--chestnut u-marginRight--20 alignItems--center">{ applyPatchErrorMessage }</span>}
                      {savePatchErr && <span className="flex flex1 u-fontSize--small u-fontWeight--medium u-color--chestnut u-marginRight--20 alignItems--center">{ savePatchErrorMessage }</span>}
                      <button type="button" disabled={dataLoading.saveKustomizeLoading || patch === "" || savingFinalize} onClick={() => this.handleKustomizeSave(false)} className="btn primary save-btn u-marginRight--normal">{dataLoading.saveKustomizeLoading && !savingFinalize ? "Saving patch" : "Save patch"}</button>
                      {patch === "" ?
                        <button type="button" onClick={this.props.skipKustomize} className="btn primary">Continue</button>
                        :
                        <button type="button" disabled={dataLoading.saveKustomizeLoading || savingFinalize} onClick={() => this.handleKustomizeSave(true)} className="btn secondary finalize-btn">{savingFinalize ? "Finalizing overlay" : "Save & continue"}</button>
                      }
                    </div>
                  }
                </div>
              </div>

            </div>
          </div>
        </div>
        <KustomizeModal
          isOpen={this.state.displayConfirmModal}
          onRequestClose={this.toggleModal}
          discardOverlay={modalAction}
          message={this.state.displayConfirmModalMessage}
          subMessage={this.state.displayConfirmModalSubMessage}
          discardMessage={this.state.displayConfirmModalDiscardMessage}
        />
      </div>
    );
  }
}