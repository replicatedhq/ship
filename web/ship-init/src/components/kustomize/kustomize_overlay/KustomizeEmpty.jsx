import React from "react";
import RenderActions from "../../shared/RenderActions";

export default class KustomizeEmpty extends React.Component {
  render() {
    const { actions, handleAction} = this.props;
    return (
      <div className="KustomizeEmpty--wrapper flex1 flex-column EmmptyState--wrapper">
        <div className="flex-1-auto u-overflow--auto container u-paddingTop--30 u-paddingBottom--30">
          <p className="u-fontSize--jumbo u-color--tuna u-fontWeight--bold u-marginBottom--normal u-lineHeight--normal">Kustomize your YAML with overlays</p>
          <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--more">
          An overlay is a target that modifies (and thus depends on) another target. The kustomization in an overlay refers to (via file path, URI or other method)
          some other kustomization, known as its base. Overlays make the most sense when there is more than one, because they create different variants of a common base
          - e.g. development, QA, staging and production environment variants.
          </p>
          <div className="product-features-wrapper">
            <div className="feature-blocks-wrapper ">
              <div className="feature-block-outer">
                <div className="feature-block-wrapper">
                  <div className="feature-block">
                    <div className="icon u-selectBaseIcon"></div>
                    <p className="title">Select a base file</p>
                    <p>Start by select a base file from the tree to the left. Files that typically use overlays are service.yaml and deployment.yaml</p>
                  </div>
                </div>
              </div>
              <div className="feature-block-outer">
                <div className="feature-block-wrapper">
                  <div className="feature-block">
                    <div className="icon u-setOverlayIcon"></div>
                    <p className="title">Create a patch</p>
                    <p>After selecting a base file, you can select specific items that you want to customize to meet your needs.</p>
                  </div>
                </div>
              </div>
              <div className="feature-block-outer">
                <div className="feature-block-wrapper">
                  <div className="feature-block">
                    <div className="icon u-shipItIcon"></div>
                    <p className="title">Ship your rendered YAML</p>
                    <p>We do the heavy lifting to merge your overlays with the base YAML and give you a single YAML file for deployment.</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
          <div className="skip-wrapper">
            <p className="u-fontSize--normal u-fontWeight--medium u-lineHeight--more">
          You are not required to customize your YAML. We built this tool to make it easy to apply overlay values and ship customized YAML quickly and effeciently. However, if you have no need to change any of these files you can move right along to the deployment step.
            </p>
            <p className="u-marginTop--20 u-fontSize--normal u-fontWeight--medium u-lineHeight--more">If you’re ready to deploy your YAML simply <span onClick={this.props.skipKustomize} className="u-color--astral u-textDecoration--underlineOnHover">click here</span>.</p>
          </div>
        </div>
        <div className="actions-wrapper container u-width--full flex flex-auto justifyContent--flexEnd">
          <RenderActions actions={actions} handleAction={handleAction} />
        </div>
      </div>
    );
  }
}
