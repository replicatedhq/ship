import React from "react";
import { mount } from "enzyme";
import { AceEditorHOC } from "./AceEditorHOC";

const validYaml = `
apiVersion: v1
kind: Service
metadata:
  name: -mytest
  labels:
    app: mytest
    chart: mytest-0.0.1
spec:
  ports:
    - name: myPort
      port: 1111
  selector:
    app: mytest
    release:
`;

describe("AceEditorHOC", () => {
    describe("provided a valid yaml", () => {
        const wrapper = mount(<AceEditorHOC fileToView={{ baseContent: "" }} />);
        it("creates markers for all values", () => {
            wrapper.setProps({
                fileToView: { baseContent: validYaml },
            });
            const markers = wrapper.state().markers;
            expect(markers).toHaveLength(9);
        });
    });
});
