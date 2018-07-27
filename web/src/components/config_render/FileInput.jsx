import React from "react";
import autoBind from "react-autobind";
import map from "lodash/map";
import after from "lodash/after";
import forEach from "lodash/forEach";

export default class FileInput extends React.Component {

  constructor(props) {
    super(props);
    this.state = {
      errText: "",
    }
    autoBind(this);
  }

  handleOnChange(ev) {
    this.setState({errText: ""});

    let files = [];
    let error;

    const done = after(ev.target.files.length, () => {
      // this.refs.file.getDOMNode().value = "";
      if (error) {
        this.setState({errText: error});
      } else if (this.props.onChange) {
        this.props.onChange(
          map(files, "value"),
          map(files, "data")
        );
      }
    });

    forEach(ev.target.files, (file) => {
      var reader = new FileReader();
      reader.onload = () => {
        var vals = reader.result.split(",");
        if (vals.length !== 2) {
          error = "Invalid file data";
        } else {
          files.push({value: file.name, data: vals[1]});
        }
        done();
      };
      reader.readAsDataURL(file);
    });
  }

  render() {

    let label;
    if (this.props.label) {
      label = this.props.label;
    } else if (this.props.multiple) {
      label = "Choose files";
    } else {
      label = "Choose file";
    }

    return (
      <span>
        <span className={`${this.props.readonly ? "readonly" : ""} ${this.props.disabled ? "disabled" : ""}`}>
          <p className="sub-header-color field-section-sub-header u-marginTop--small u-marginBottom--small">{label}</p>
          <input
            ref="file"
            type="file"
            name={this.props.name}
            className="form-control"
            onChange={this.handleOnChange}
            readOnly={this.props.readOnly}
            multiple={this.props.multiple}
            disabled={this.props.disabled} />
        </span>
        <small className="text-danger"> {this.state.errText}</small>
      </span>
    );
  }
}
