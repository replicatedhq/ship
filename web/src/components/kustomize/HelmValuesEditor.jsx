import React from "react";
import autoBind from "react-autobind";
import * as linter from "replicated-lint";
import Linter from "../shared/Linter";
import AceEditor from "react-ace";
import ErrorBoundary from "../../ErrorBoundary";
import Toast from "../shared/Toast";
import get from "lodash/get";
import find from "lodash/find";

import "../../../node_modules/brace/mode/yaml";
import "../../../node_modules/brace/theme/chrome";

import "../../scss/components/kustomize/HelmValuesEditor.scss";

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
      unsavedChanges: false,
      toastDetails: {
        opts: {}
      }
    }
    autoBind(this);
  }

  componentDidMount() {
    if(this.props.getStep.values) {
      this.setState({
        initialSpecValue: this.props.getStep.values,
        specValue: this.props.getStep.values
      })
    }
  }

  getLinterErrors(specContents) {
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

  onSpecChange(value) {
    const { initialSpecValue } = this.state;
    this.getLinterErrors(value);
    this.setState({
      specValue: value,
      unsavedChanges: !(initialSpecValue === value)
    });
  }

  cancelToast() {
    let nextState = {};
    nextState.toastDetails = {
      showToast: false,
      title: "",
      subText: "",
      type: "default",
      opts: {}
    };
    this.setState(nextState)
  }

  handleContinue() {
    const { actions, isNewRouter } = this.props;
    let submitAction;
    if (isNewRouter) {
      submitAction = find(actions, ["buttonType", "popover"]);
    } else {
      submitAction = find(actions, ["sort", 1]);
    }
    this.props.handleAction(submitAction);
  }

  onValuesSaved() {
    let nextState = {};
    nextState.toastDetails = {
      showToast: true,
      title: "All changes have been saved.",
      type: "default",
      opts: {
        showCancelButton: true,
        confirmButtonText: "Continue to next step",
        confirmAction: () => { this.handleContinue(); }
      }
    }
    this.setState(nextState);
  }

  handleSkip() {
    const { initialSpecValue } = this.state;
    const payload = {
      values: initialSpecValue
    }
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

  handleSaveValues() {
    const { specValue, initialSpecValue } = this.state;
    const payload = {
      values: specValue
    }
    if(payload.values !== "" && payload.values !== initialSpecValue) {
      this.setState({ saving: true, helmLintErrors: [] });
      this.props.saveValues(payload)
        .then(({ errors }) => {
          if (errors) {
            return this.setState({
              saving: false,
              helmLintErrors: errors
            });
          }
          this.setState({ saving: false, savedYaml: true });
          this.onValuesSaved();
        })
        .catch((err) => {
          // TODO: better handling
          console.log(err);
        })
    }
  }

  render() {
    const {
      readOnly,
      specValue,
      saving,
      toastDetails,
      unsavedChanges,
      helmLintErrors,
    } = this.state;
    const {
      values,
      readme,
      name
    } = this.props.helmChartMetadata;

    return (
      <ErrorBoundary>
        <div className="flex-column flex1 HelmValues--wrapper">
          <Toast toast={toastDetails} onCancel={this.cancelToast} />
          <p className="u-color--dutyGray u-fontStize--large u-fontWeight--medium u-marginBottom--small">
            /{name}/
            <span className="u-color--tuna u-fontWeight--bold">values.yaml</span>
          </p>
          <p className="u-color--dustyGray u-fontSize--normal u-marginTop--normal u-marginBottom--20">Here you can edit the values.yaml to specify values for your application. You will be able to apply overlays for your YAML in the next step.</p>
          <div className="AceEditor--wrapper helm-values flex1 flex u-height--full u-width--full u-overflow--hidden u-marginBottom--20">
            <div className="flex1 flex-column u-width--half">
              <AceEditor
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
              <Linter errors={this.state.specErrors} spec={values} previewEnabled={true} readme={readme} />
            </div>
          </div>
          <div className="action container u-width--full u-marginTop--30 flex flex1 justifyContent--flexEnd u-position--fixed u-bottom--0 u-right--0 u-left--0">
            <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-marginRight--30 u-lineHeight--normal">{helmLintErrors.join("\n")}</p>
            <div className="flex flex-auto alignItems--center">
              <p
                className="u-color--astral u-fontSize--normal u-fontWeight--medium u-marginRight--20 u-cursor--pointer"
                onClick={() => { this.handleSkip() }}>
                Skip this step
              </p>
              <button
                className="btn primary"
                onClick={() => this.handleSaveValues()}
                disabled={saving || !unsavedChanges}>
                {saving ? "Saving" : "Save values"}
              </button>
            </div>
          </div>
        </div>
      </ErrorBoundary>
    );
  }
}
