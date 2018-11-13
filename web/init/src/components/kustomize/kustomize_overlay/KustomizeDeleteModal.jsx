import React from "react";
import Modal from "react-modal";
import PropTypes from "prop-types";

const KustomizeDeleteModal = ({
    isOpen,
    onRequestClose,
    discardOverlay,
    message,
    discardMessage,
}) => (
    <Modal
    isOpen={isOpen}
    onRequestClose={onRequestClose}
    shouldReturnFocusAfterClose={false}
    ariaHideApp={false}
    contentLabel="Modal"
    className="Modal DefaultSize"
  >
    <div className="Modal-header">
      <p>{ message }</p>
    </div>
    <div className="flex flex-column Modal-body">
      <p className="u-fontSize--large u-fontWeight--normal u-color--dustyGray u-lineHeight--more">It will not be applied to the kustomization.yaml file that is generated for you.</p>
      <div className="flex justifyContent--flexEnd u-marginTop--20">
        <button className="btn secondary u-marginRight--10" onClick={() => onRequestClose("")}>Cancel</button>
        <button type="button" className="btn primary" onClick={discardOverlay}>{discardMessage}</button>
      </div>
    </div>
  </Modal>
);

KustomizeDeleteModal.propTypes = {
    // boolean to control whether the modal is open
    isOpen: PropTypes.bool,
    onRequestClose: PropTypes.func,
    discardOverlay: PropTypes.func,
    // message to display in the body of the modal
    message: PropTypes.string,
    // text to display within the delete button
    discardMessage: PropTypes.string,
}

export default KustomizeDeleteModal;
