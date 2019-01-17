import React from "react";

export class AceEditorHOC extends React.Component {
    constructor(props) {
        super(props);

        this.editor = {
            resize: () => {},
        };
    }

    render() {
        return (
            <div>AceEditorHOCMock</div>
        )
    }
}
