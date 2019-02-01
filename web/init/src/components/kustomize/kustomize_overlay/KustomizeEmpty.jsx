import React from "react";
import RenderActions from "../../shared/RenderActions";

export default class KustomizeEmpty extends React.Component {
  render() {
    const { actions, handleAction , goBack, firstRoute } = this.props;
    return (
      <div className="KustomizeEmpty--wrapper u-paddingTop--30 flex1 flex-column justifyContent--spaceBetween">
        <div className="flex-column flex-1-auto u-overflow--auto container">
          <div className="HelmIntro--wrapper flex-column">
            <p className="u-fontSize--jumbo2 u-color--tuna u-fontWeight--bold u-lineHeight--normal">Kustomize your YAML</p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--more">Ship has generated all of the Kubernetes YAML from the Kubernetes manifests and has prepared the application for deployment to a cluster. On the next screen, you’ll see a file tree showing all of the Kubernetes resources. You can review them, and click on any line to create a patch using Kustomize and edit (or add/remove).</p>
            <div className="HelmIntro--diagram flex">
              <div className="values-step flex-column justifyContent--center u-position--relative">
                <div className="icon checkmark"></div>
                <div className="flex flex-column alignItems--center">
                  <div className="icon manifest-small"></div>
                  <p className="u-textAlign--center u-fontSize--small u-color--tuna u-fontWeight--bold u-marginTop--normal u-lineHeight--small">Kubernetes Manifests</p>
                </div>
              </div>
              <div className="border-wth-arrow flex flex-column alignItems--center">
                <div className="line flex1"></div>
                <div className="icon arrow flex-auto"></div>
                <div className="line flex1"></div>
              </div>
              <div className="detailed-steps flex flex-column">
                <div className="detailed-step flex">
                  <div className="icon base flex-auto"></div>
                  <div className="flex flex-column">
                    <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Base</p>
                    <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--more u-fontWeight--medium">The base is the rendered Kubernetes manifests. Ship will create this for you and it’s generated from the original manifests, never forked.</p>
                  </div>
                </div>
                <div className="detailed-step flex">
                  <div className="icon patches flex-auto"></div>
                  <div className="flex flex-column">
                    <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Patches</p>
                    <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--more u-fontWeight--medium">The changes you would have made in a fork, i.e. any advanced customization (additions, deletions or changes) to the base can be written as patches and will be managed using <a href="http://kustomize.io" target="_blank" rel="noopener noreferrer" className="u-color--astral u-fontWeight--medium">Kustomize</a>. Ship will guide you through creating these patches.</p>
                  </div>
                </div>
              </div>
              <div className="border-wth-es flex flex-column alignItems--center">
                <div className="line flex1"></div>
                <p className="plus u-color--chateauGreen u-fontSize--jumbo2 u-fontWeight--bold">=</p>
                <div className="line flex1"></div>
              </div>
              <div className="deployment-step flex alignItems--center">
                <div className="flex flex-column alignItems--center">
                  <div className="icon deployable-app"></div>
                  <p className="u-textAlign--center u-fontSize--small u-color--tuna u-fontWeight--bold u-marginTop--normal u-lineHeight--small">Deployable App</p>
                </div>
              </div>
            </div>
            <div className="flex flex-column flex1 u-borderTop--gray">
              <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--more">Ship will keep all of your patches separate from the upstream (base) YAML. This allows Ship to pull the latest version of the application every time it’s updated and merge your patches back in.</p>
              <p className="u-marginTop--20 u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--more">To continue, click Next and review some YAML.</p>
            </div>
          </div>
        </div>
        <div className="actions-wrapper container u-width--full flex flex-auto">
          {firstRoute ? null :
            <div className="flex-auto u-marginRight--normal">
              <button className="btn secondary" onClick={() => goBack()}>Back</button>
            </div>
          }
          <div className="flex1 flex justifyContent--flexEnd">
            <RenderActions actions={actions} handleAction={handleAction} />
          </div>
        </div>
      </div>
    );
  }
}
