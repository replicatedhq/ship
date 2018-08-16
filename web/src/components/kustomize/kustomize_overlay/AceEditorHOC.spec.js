import React from "react";
import { mount } from "enzyme";
import { AceEditorHOC } from "./AceEditorHOC";

const testYaml = `
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
    describe("provided a valid yaml that is supported", () => {
        const wrapper = mount(<AceEditorHOC fileToView={{ baseContent: "" }} />);
        it("creates markers for all values", () => {
            wrapper.setProps({
                fileToView: { baseContent: testYaml, isSupported: true },
            });
            const markers = wrapper.state().markers;
            expect(markers).toHaveLength(9);
        });
    });
    describe("provided an unsupported yaml", () => {
        const wrapper = mount(<AceEditorHOC fileToView={{ baseContent: "" }} />);
        it("creates markers for all values", () => {
            wrapper.setProps({
                fileToView: { baseContent: testYaml, isSupported: false },
            });
            const markers = wrapper.state().markers;
            expect(markers).toHaveLength(0);
        });
    });
});
