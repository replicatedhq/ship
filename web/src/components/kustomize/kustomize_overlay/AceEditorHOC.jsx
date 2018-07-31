import React from "react";
import ace from "brace";
import AceEditor from "react-ace";
import * as ast from "yaml-ast-parser";
import find from "lodash/find";

const { addListener } = ace.acequire("ace/lib/event");

export default class AceEditorHOC extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      activeMarker: [],
      markers: [],
    };
  }

  componentDidUpdate(prevProps) {
    const { fileToView } = this.props;
    if (fileToView) {
      const { fileToView: oldFileToView = {} } = prevProps;
      const { baseContent: oldBaseContent } = oldFileToView;
      const { baseContent } = fileToView;
      if (baseContent !== oldBaseContent) {
        const markers = this.createMarkers(fileToView);
        this.setState({ markers });
      }
    }
  }

  componentDidMount() {
    addListener(this.aceEditorBase.editor, "click", this.addToOverlay)
    addListener(this.aceEditorBase.editor.renderer.scroller, "mousemove", this.setActiveMarker);
    addListener(this.aceEditorBase.editor.renderer.scroller, "mouseout", this.setActiveMarker);
  }

  findMarkerAtRow = (row, markers) => (
    find(markers, ({ startRow, endRow }) => ( row >= startRow && row <= endRow ))
  )

  addToOverlay = (e) => {
    const { markers } = this.state;
    const { row } = e.getDocumentPosition();
    const matchingMarker = this.findMarkerAtRow(row, markers);
    console.log("matching", matchingMarker);
    if (matchingMarker) {
      const overlayKeyValue = matchingMarker.mapping.key.value;
      this.props.addToOverlay(overlayKeyValue);
    }
  }

  setActiveMarker = (e) => {
    const { markers, activeMarker } = this.state;
    const { clientX, clientY } = e;
    const { row } = this.aceEditorBase.editor.renderer.screenToTextCoordinates(clientX, clientY);
    const matchingMarker = this.findMarkerAtRow(row, markers);

    if (matchingMarker) {
      const [ activeMarker0 = {} ] = activeMarker;
      if (matchingMarker.startRow !== activeMarker0.startRow) {
        this.setState({ activeMarker: [ matchingMarker ] });
      }
    }
  }

  createMarkers = (fileToView) => {
    if (this.aceEditorBase) {
      const aceDoc = this.aceEditorBase.editor.getSession().getDocument();
      const loadedAst = ast.safeLoad(fileToView.baseContent, null);
      console.log("loadedAst", loadedAst);

      return loadedAst.mappings.map((mapping) => {
        const { value } = mapping;
        const { startPosition, endPosition } = value;
        const { row: startRow } = aceDoc.indexToPosition(startPosition, 0);
        const { row: endRow } = aceDoc.indexToPosition(endPosition, 0);

        return {
          startRow,
          endRow: endRow + 1,
          className: "test",
          mapping,
        };
      });
    }
    return [];
  }

  render() {
    const { fileToView } = this.props;
    const { activeMarker } = this.state;
    console.log("HOWMANYTIMESAMIRENDERING");

    return (
      <AceEditor
        ref={(editor) => { this.aceEditorBase = editor }}
        mode="yaml"
        theme="chrome"
        className="flex1 flex disabled-ace-editor ace-chrome"
        readOnly={true}
        value={fileToView && fileToView.baseContent || ""}
        height="100%"
        width="100%"
        editorProps={{
          $blockScrolling: Infinity,
          useSoftTabs: true,
          tabSize: 2,
        }}
        setOptions={{
          scrollPastEnd: false
        }}
        markers={activeMarker}
      />
    );
  }
}