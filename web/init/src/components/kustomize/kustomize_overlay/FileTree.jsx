import * as React from "react";

export default class FileTree extends React.Component {

  handleFileSelect = (path) => {
    this.props.handleFileSelect(path);
  }

  handleDeleteOverlay = (e, path) => {
    e.stopPropagation();
    this.props.handleDeleteOverlay(path);
  }

  render() {
    const { files, basePath, isRoot, selectedFile, handleFileSelect, handleDeleteOverlay, isOverlayTree, isResourceTree } = this.props;
    return (
      <ul className={`${isRoot ? "FileTree-wrapper" : "u-marginLeft--normal"}`}>
        {files && files.map((file, i) => ( file.children && file.children.length ?
          <li key={`${file.path}-Directory-${i}`} className={`u-position--relative u-userSelect--none ${file.hasOverlay && "edited"}`}>
            <input type="checkbox" name={`sub-dir-${file.name}-${file.children.length}-${file.path}-${basePath}-${i}`} id={`sub-dir-${file.name}-${file.children.length}-${file.path}-${basePath}-${i}`} defaultChecked={true} />
            <label htmlFor={`sub-dir-${file.name}-${file.children.length}-${file.path}-${basePath}-${i}`}>{file.name === "/" ? basePath : file.name}</label>
            <FileTree
              files={file.children}
              handleFileSelect={(path) => handleFileSelect(path)}
              handleDeleteOverlay={(path) => handleDeleteOverlay(path)}
              selectedFile={selectedFile}
              isOverlayTree={isOverlayTree}
            />
          </li>
          :
          <li key={file.path} className={`u-position--relative is-file ${selectedFile === file.path ? "is-selected" : ""} ${file.hasOverlay ? "edited" : ""}`} onClick={() => this.handleFileSelect(file.path)}>
            {file.name}
            {isOverlayTree || isResourceTree ? <span className="icon clickable u-deleteOverlayIcon" onClick={(e) => this.handleDeleteOverlay(e, file.path)}></span> : null}
          </li>
        ))
        }
      </ul>
    );
  }
}