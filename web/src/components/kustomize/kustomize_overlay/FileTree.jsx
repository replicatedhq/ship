import * as React from "react";
import autoBind from "react-autobind";

export default class FileTree extends React.Component {

  constructor() {
    super();
    autoBind(this);
  }

  handleFileSelect(path) {
    this.props.handleFileSelect(path);
  }

  render() {
    const { files, basePath, isRoot, selectedFile, handleFileSelect } = this.props;
    return (
      <ul className={`${isRoot ? "FileTree-wrapper" : "u-marginLeft--normal"}`}>
        {files && files.map((file, i) => (
          file.children && file.children.length ?
            <li key={`${file.path}-Directory-${i}`} className={`u-position--relative u-userSelect--none ${file.hasOverlay && "edited"}`}>
              <input type="checkbox" name={`sub-dir-${file.name}-${file.children.length}-${file.path}-${i}`} id={`sub-dir-${file.name}-${file.children.length}-${file.path}-${i}`} defaultChecked={true} />
              <label htmlFor={`sub-dir-${file.name}-${file.children.length}-${file.path}-${i}`}>{file.name === "/" ? basePath : file.name}</label>
              <FileTree
                files={file.children}
                handleFileSelect={(path) => handleFileSelect(path)}
                selectedFile={selectedFile}
              />
            </li>
            :
            <li key={file.path} className={`u-position--relative is-file ${selectedFile === file.path ? "is-selected" : ""} ${file.hasOverlay ? "edited" : ""}`} onClick={() => this.handleFileSelect(file.path)}>{file.name}</li>
        ))
        }
      </ul>
    );
  }
}
