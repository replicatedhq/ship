import React from "react";
import ace from "brace";
import AceEditor from "react-ace";
import * as yaml from "js-yaml";
import * as ast from "yaml-ast-parser";
import find from "lodash/find";
import set from "lodash/set";

const { addListener, removeListener } = ace.acequire("ace/lib/event");

export const PATCH_TOKEN = "TO_BE_MODIFIED";

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
    if (fileToView !== prevProps.fileToView) {
      if (fileToView.baseContent && fileToView.isSupported) {
        const markers = this.createMarkers(fileToView);
        this.setState({ markers });
      }
    }
    if (
      (this.props.overlayOpen !== prevProps.overlayOpen) ||
      (this.props.diffOpen !== prevProps.diffOpen)
    ) {
      if (this.aceEditorBase) {
        this.aceEditorBase.editor.resize();
      }
    }
  }

  componentDidMount() {
    addListener(this.aceEditorBase.editor, "click", this.addToOverlay)
    addListener(this.aceEditorBase.editor.renderer.scroller, "mousemove", this.setActiveMarker);
    addListener(this.aceEditorBase.editor.renderer.scroller, "mouseout", this.setActiveMarker);
  }

  componentWillUnmount() {
    removeListener(this.aceEditorBase.editor, "click", this.addToOverlay);
    removeListener(this.aceEditorBase.editor.renderer.scroller, "mousemove", this.setActiveMarker);
    removeListener(this.aceEditorBase.editor.renderer.scroller, "mouseout", this.setActiveMarker)
  }

  findMarkerAtRow = (row, markers) => (
    find(markers, ({ startRow, endRow }) => ( row >= startRow && row <= endRow ))
  )

  addToOverlay = () => {
    const { fileToView } = this.props;
    const { activeMarker } = this.state;

    if (activeMarker.length > 0) {
      const matchingMarker = activeMarker[0];
      const { path } = matchingMarker;

      let tree = yaml.safeLoad(fileToView.baseContent);
      const modifiedTree = set(tree, matchingMarker.path, PATCH_TOKEN);
      const dirtybaseContent = yaml.safeDump(modifiedTree);
      this.props.handleGeneratePatch(dirtybaseContent, path);
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
      this.createMarkersRec(loadedAst, [], markers);
      return markers;
    }
  }

  createMarkersRec = (ast, path, markers) => {
    const aceDoc = this.aceEditorBase.editor.getSession().getDocument();

    const createMarkersSlice = ({ items }) => {
      if (items && items.length > 0) {
        for (let i = 0; i < items.length; i++) {
          const item = items[i];
          this.createMarkersRec(item, [...path, i], markers);
        }
      }
    };

    const createMarkersMap = ({ mappings }) => {
      for (const mapping of mappings) {
        const { value, key } = mapping;
        if (value === null) {
          const { startPosition, endPosition } = ast;
          const { row: startRow } = aceDoc.indexToPosition(startPosition, 0);
          const { row: endRow } = aceDoc.indexToPosition(endPosition, 0);
          const nullMarker = {
            startRow,
            endRow: endRow + 1,
            className: "marker-highlight-null",
            mapping,
            path: [...path, key.value],
          }
          return markers.push(nullMarker);
        }
        const newPathKey = key.value;
        this.createMarkersRec(value, [...path, newPathKey], markers);
      }
    };

    if (ast.mappings) {
      return createMarkersMap(ast);
    }
    if (ast.items) {
      return createMarkersSlice(ast);
    }

    const { startPosition, endPosition } = ast;
    const { row: startRow } = aceDoc.indexToPosition(startPosition, 0);
    const { row: endRow } = aceDoc.indexToPosition(endPosition, 0);
    const newMarker = {
      startRow,
      endRow: endRow + 1,
      className: "marker-highlight",
      mapping: ast,
      path,
    };
    markers.push(newMarker);
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
