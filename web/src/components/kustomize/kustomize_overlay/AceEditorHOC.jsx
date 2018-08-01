import React from "react";
import ace from "brace";
import AceEditor from "react-ace";
import * as ast from "yaml-ast-parser";
import find from "lodash/find";

const { addListener } = ace.acequire("ace/lib/event");

export const PATCH_TOKEN = "TO_BE_MODIFIED";
const YAML_LIST_PREFIX = "-";

export class AceEditorHOC extends React.Component {
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
    const getPrefix = (value) => {
      if (value.mapping.parent.key) {
        return `${value.mapping.parent.key.rawValue}: `;
      }
      return `${YAML_LIST_PREFIX} `;
    };

    const { activeMarker } = this.state;

    if (activeMarker.length > 0) {
      const matchingMarker = activeMarker[0];
      if (matchingMarker.mapping.value) {
        const valueToEdit = matchingMarker.mapping.rawValue;
        const prefix = getPrefix(matchingMarker);
        const baseContent = this.aceEditorBase.editor.getValue();
        const dirtybaseContent = baseContent.replace(`${prefix}${valueToEdit}`, `${prefix}${PATCH_TOKEN}`);
        this.props.handleGeneratePatch(dirtybaseContent);
      }
    }
  }

  setActiveMarker = (e) => {
    const { clientY } = e;
    const { markers, activeMarker } = this.state;

    const renderer = this.aceEditorBase.editor.renderer;
    const canvasPos = renderer.scroller.getBoundingClientRect();

    const row = Math.ceil((clientY + renderer.scrollTop - canvasPos.top) / renderer.lineHeight);
    const matchingMarker = this.findMarkerAtRow(row, markers);

    if (matchingMarker) {
      renderer.setCursorStyle("pointer");
      const [ activeMarker0 = {} ] = activeMarker;
      if (matchingMarker.startRow !== activeMarker0.startRow) {
        this.setState({ activeMarker: [ matchingMarker ] });
      }
    } else {
      this.setState({ activeMarker: [] });
    }
  }

  createMarkers = (fileToView) => {
    if (this.aceEditorBase) {
      let markers = [];
      const loadedAst = ast.safeLoad(fileToView.baseContent, null);
      this.createMarkersRec(loadedAst, markers);
      return markers;
    }
  }

  createMarkersRec = (ast, markers) => {
    const aceDoc = this.aceEditorBase.editor.getSession().getDocument();

    if (!ast.mappings) {
      if (ast.items && ast.items.length > 0) {
        for (const item of ast.items) {
          this.createMarkersRec(item, markers);
        }
      }
      else {
        const { startPosition, endPosition } = ast;
        const { row: startRow } = aceDoc.indexToPosition(startPosition, 0);
        const { row: endRow } = aceDoc.indexToPosition(endPosition, 0);
        const newMarker = {
          startRow,
          endRow: endRow + 1,
          className: "marker-highlight",
          mapping: ast,
        };
        markers.push(newMarker);
      }
      return;
    }

    for (const mapping of ast.mappings) {
      if (mapping.value === null) {
        const { startPosition, endPosition } = ast;
        const { row: startRow } = aceDoc.indexToPosition(startPosition, 0);
        const { row: endRow } = aceDoc.indexToPosition(endPosition, 0);
        const nullMarker = {
          startRow,
          endRow: endRow + 1,
          className: "marker-highlight-null",
          mapping,
        }
        return markers.push(nullMarker);
      }

      this.createMarkersRec(mapping.value, markers);
    }
  }

  render() {
    const { fileToView } = this.props;
    const { activeMarker } = this.state;

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