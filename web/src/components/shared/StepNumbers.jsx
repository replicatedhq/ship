import * as React from "react";
import { withRouter } from "react-router-dom";
import isEmpty from "lodash/isEmpty";
import find from "lodash/find";
import indexOf from "lodash/indexOf";
import "../../scss/components/shared/StepNumbers.scss";

class StepNumbers extends React.Component {
  constructor() {
    super();
    this.state = {
      currentStep: 0,
      progressLength: 0,
      steps: [],
    }
  }

  getPositionAtCenter(element) {
    const data = element.getBoundingClientRect();
    return {
      x: data.left + data.width / 2,
      y: data.top + data.height / 2
    };
  }

  getDistanceBetweenElements(a, b) {
    var aPosition = this.getPositionAtCenter(a);
    var bPosition = this.getPositionAtCenter(b);
    return Math.sqrt(Math.pow(aPosition.x - bPosition.x, 2) + Math.pow(aPosition.y - bPosition.y, 2));
  }

  setStepsToState() {
    const { steps } = this.props;
    const stateSteps = steps.map((step) => {
      const cleanedPath = this.props.location.pathname.split("/")[1];
      const newStep = {
        ...step,
        isComplete: false,
        isActive: cleanedPath === step.id,
      };
      return newStep;
    });
    const currIdx = find(stateSteps, ["isActive", true]);
    const currStep = indexOf(stateSteps, currIdx);
    this.setState({ steps: stateSteps, currentStep: currStep });
    this.setCompleteSteps(currStep, stateSteps);
  }

  setCompleteSteps(currentStep, steps) {
    for (let i = 0; i < currentStep; i++) {
      let currStep = steps[i];
      currStep.isComplete = true;
    }
    this.setState({ steps });
  }

  determineCurrentStep(id) {
    let stateStep = find(this.state.steps, ["id", id]);
    const stateStepIndex = indexOf(this.state.steps, stateStep);
    const { currentStep } = this.state;
    stateStep.isActive = currentStep === stateStepIndex ? true : false;
  }

  goToStep(idx) {
    const step = this.state.steps[idx];
    this.props.history.push(`/${step.id}`);
  }

  componentDidUpdate(lastProps, lastState) {
    if (
      (this.props.location.pathname !== lastProps.location.pathname) ||
      (this.props.steps !== lastProps.steps && !isEmpty(this.props.steps))
    ) {
      this.setStepsToState();
    }
    if (this.state.currentStep !== lastState.currentStep) {
      const elOne = find(this.state.steps, ["isComplete", true]);
      const elTwo = find(this.state.steps, ["isActive", true]);
      if (elOne && elTwo) {
        const length = this.getDistanceBetweenElements(document.getElementById(elOne.id), document.getElementById(elTwo.id));
        this.setState({ progressLength: length });
      }
    }
  }

  componentDidMount() {
    if (!isEmpty(this.props.steps)) {
      this.setStepsToState();
    }
  }

  renderSteps() {
    const { steps } = this.state;
    if (!steps.length) return;
    const renderedSteps = this.state.steps.map((step, i) => {
      this.determineCurrentStep(step.id); // Is this the current step, if so set to active
      return (
        <div key={`${step.id}-${i}`} id={step.id} className={`flex-auto u-cursor--pointer flex step-number ${step.isActive ? "is-active" : ""} ${step.isComplete ? "is-complete" : ""}`} onClick={() => this.goToStep(i)}>
          <span className="number flex-column flex-verticalCenter alignItems--center">{step.isComplete ? <span className="icon u-smallCheckWhite"></span> : i + 1}</span>
        </div>
      )
    });
    return renderedSteps;
  }

  render() {
    const { currentStep, progressLength } = this.state;
    return (
      <div className="flex-column flex-auto">
        {!isEmpty(this.state.steps) ? <div className="steps-numbers-wrapper">
          <div className="numbers-wrapper flex flex1 justifyContent--spaceBetween">
            {this.renderSteps()}
            {currentStep > 0 && <span className="completed-progress-bar" style={{ width: `${progressLength}px` }}></span>}
            <span className="progress-base"></span>
          </div>
        </div>
          : null}
      </div>
    );
  }
}

export default withRouter(StepNumbers);