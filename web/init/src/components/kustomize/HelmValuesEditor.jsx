import React from "react";
import * as linter from "replicated-lint";
import Linter from "../shared/Linter";
import AceEditor from "react-ace";
import ErrorBoundary from "../../ErrorBoundary";
import HelmAdvancedInput from "./HelmAdvancedInput";
import get from "lodash/get";
import find from "lodash/find";

import "../../../node_modules/brace/mode/yaml";
import "../../../node_modules/brace/theme/chrome";
import "../../../node_modules/brace/ext/searchbox";

export default class HelmValuesEditor extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      readOnly: false,
      showConsole: false,
      specErrors: [],
      specValue: "",
      initialSpecValue: "",
      helmLintErrors: [],
      saving: false,
      saveFinal: false,
      unsavedChanges: false,
      initialHelmReleaseName: "",
      initialHelmNamespace: "",
      helmReleaseName: "",
      displaySettings: false,
      helmNamespace: "",
    }
  }

  componentDidMount() {
    window.addEventListener("keydown", this.handleKeyboardSave);
    if (this.props.getStep.values) {
      this.setState({
        initialSpecValue: this.props.getStep.values,
        specValue: this.props.getStep.values,
        initialHelmReleaseName: this.props.getStep.helmName,
        initialHelmNamespace: this.props.getStep.namespace,
        helmReleaseName: this.props.getStep.helmName,
        helmNamespace: this.props.getStep.namespace,
      });
      this.helmEditor.editor.getSession().setValue(this.props.getStep.values); // this resets the UndoManager and prevents the editor from being able to be wiped out by too many CMD+Z's
    }
  }



  getLinterErrors = (specContents) => {
    if (specContents === "") return;
    const errors = new linter.Linter(specContents).lint();
    let markers = [];
    for (let error of errors) {
      if (error.positions) {
        for (let position of error.positions) {
          let line = get(position, "start.line");
          if (line) {
            markers.push({
              startRow: line,
              endRow: line + 1,
              className: "error-highlight",
              type: "background"
            })
          }
        }
      }
    }
    this.setState({
      specErrors: errors,
      specErrorMarkers: markers,
    });
  }

  onSpecChange = (value) => {
    const { initialSpecValue } = this.state;
    this.getLinterErrors(value);
    this.setState({
      specValue: value,
      unsavedChanges: !(initialSpecValue === value)
    });
  }

  handleContinue = () => {
    const { actions } = this.props;
    let submitAction;
    submitAction = find(actions, ["buttonType", "popover"]);
    this.props.handleAction(submitAction, true);
  }

  handleSkip = () => {
    const { initialSpecValue } = this.state;
    const payload = {
      values: initialSpecValue
    };
    this.setState({ helmLintErrors: [] });
    this.props.saveValues(payload)
      .then(({ errors }) => {
        if (errors) {
          return this.setState({
            saving: false, helmLintErrors: errors
          });
        }
        this.handleContinue();
      })
      .catch((err) => {
        // TODO: better handling
        console.log(err);
      })
  }

  handleSaveValues = (finalize) => {
    const { specValue, helmReleaseName, helmNamespace } = this.state;
    const payload = {
      values: specValue,
      releaseName: helmReleaseName,
      namespace: helmNamespace,
    };
    if (payload.values !== "") {
      this.setState({ saving: true, savedYaml: false, saveFinal: finalize, helmLintErrors: [] });
      this.props.saveValues(payload)
        .then(({ errors }) => {
          if (errors) {
            return this.setState({
              saving: false,
              helmLintErrors: errors,
              saveFinal: false,
            });
          }
          this.setState({ saving: false, savedYaml: true });
          if (finalize) {
            this.handleContinue();
          } else {
            setTimeout(() => {
              if (this.helmEditor) {
                this.setState({ savedYaml: false });
              }
            }, 3000);
          }
        })
        .catch((err) => {
          // TODO: better handling
          console.log(err);
        })
    }
  }

  handleKeyboardSave = (e) => {
    const sKey = e.keyCode === 83;
    if (sKey && (e.ctrlKey || e.metaKey)) {
      e.preventDefault();
      e.stopPropagation();
      this.handleSaveValues(false);
    }
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.handleKeyboardSave);
  }

  handleOnChangehelmReleaseName = (helmReleaseName) => this.setState({ helmReleaseName })

  handleOnChangehelmNamespace = (helmNamespace) => this.setState({ helmNamespace })

  render() {
    const {
      readOnly,
      specValue,
      initialSpecValue,
      saving,
      saveFinal,
      savedYaml,
      helmLintErrors,
      initialHelmReleaseName,
      helmReleaseName,
      displaySettings,
      helmNamespace,
      initialHelmNamespace,
    } = this.state;
    const {
      values,
      readme,
      name
    } = this.props.shipAppMetadata;
    const { firstRoute, goBack } = this.props;
    return (
      <ErrorBoundary>
        <div className="flex-column flex1 HelmValues--wrapper u-paddingTop--30">
          <div className="flex-column flex-1-auto u-overflow--auto container">
            <p className="u-color--dutyGray u-fontStize--large u-fontWeight--medium u-marginBottom--small">
            /{name}/
              <span className="u-color--tuna u-fontWeight--bold">values.yaml</span>
            </p>
            <p className="u-color--dustyGray u-fontSize--normal u-marginTop--normal u-marginBottom--20">Here you can edit the values.yaml to specify values for your application. You will be able to apply overlays for your YAML in the next step.</p>
            <div className="advanced-settings-wrapper u-marginTop--10">
              <div className={`section-border ${displaySettings ? "open" : "closed"} flex justifyContent--center u-position--relative`}>
                <p className="flex-auto u-fontSize--small u-color--tundora u-fontWeight--medium u-cursor--pointer" onClick={() => { this.setState({ displaySettings: !this.state.displaySettings }) }}>{displaySettings ? "Close Advanced Settings" : "Show Advanced Settings"}</p>
              </div>
              {displaySettings ? <div className="settings u-marginBottom--20">
                <HelmAdvancedInput
                  value={helmReleaseName}
                  onChange={this.handleOnChangehelmReleaseName}
                  title="Helm Name"
                  subTitle="This is the name that will be used to template your Helm chart."
                />
                <HelmAdvancedInput
                  value={helmNamespace}
                  onChange={this.handleOnChangehelmNamespace}
                  title="Helm Namespace"
                  subTitle="This is the namespace that will be used to template your Helm chart."
                  placeholder="default"
                />
              </div> : null}
            </div>
            <div className="AceEditor--wrapper helm-values flex1 flex u-height--full u-width--full">
              <div className="flex1 flex-column u-width--half">
                <AceEditor
                  ref={(editor) => { this.helmEditor = editor }}
                  mode="yaml"
                  theme="chrome"
                  className={`${readOnly ? "disabled-ace-editor ace-chrome" : ""}`}
                  readOnly={readOnly}
                  onChange={this.onSpecChange}
                  markers={this.state.specErrorMarkers}
                  value={specValue}
                  height="100%"
                  width="100%"
                  editorProps={{
                    $blockScrolling: Infinity,
                    useSoftTabs: true,
                    tabSize: 2,
                  }}
                  setOptions={{
                    scrollPastEnd: true
                  }}
                />
              </div>
              <div className={`flex-auto flex-column console-wrapper u-width--third ${!this.state.showConsole ? "visible" : ""}`}>
                <Linter errors={this.state.specErrors} spec={values} previewEnabled={true} readme={this.props.getStep.readme || readme} />
              </div>
            </div>
          </div>
          <div className="actions-wrapper container u-width--full flex flex-auto">
            {firstRoute ? null :
              <div className="flex-auto u-marginRight--normal">
                <button className="btn secondary" onClick={() => goBack()}>Back</button>
              </div>
            }
            <div className="flex flex1 alignItems--center justifyContent--flexEnd">
              {initialSpecValue === specValue && initialHelmReleaseName === helmReleaseName && helmNamespace === initialHelmNamespace ?
                <button className="btn primary" onClick={() => { this.handleSkip() }}>Continue</button>
                :
                <div className="flex">
                  <div className="flex1 flex-column flex-verticalCenter">
                    {helmLintErrors.length ?
                      <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginRight--normal u-lineHeight--normal">{helmLintErrors.join("\n")}</p>
                    : null}
                    {savedYaml ?
                      <p className="u-color--vidaLoca u-fontSize--small u-fontWeight--medium u-marginRight--normal u-lineHeight--normal">Values saved</p>
                    : null}
                  </div>
                  <button className="btn primary u-marginRight--normal" onClick={() => this.handleSaveValues(false)} disabled={saving || saveFinal}>{saving ? "Saving" : "Save values"}</button>
                  <button className="btn secondary" onClick={() => this.handleSaveValues(true)} disabled={saving || saveFinal}>{saveFinal ? "Saving values" : "Save & continue"}</button>
                </div>
              }
            </div>
          </div>

        </div>
      </ErrorBoundary>
    );
  }
}
