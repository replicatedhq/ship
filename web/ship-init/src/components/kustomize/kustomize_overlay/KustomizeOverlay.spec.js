import React from "react";
import { mount } from "enzyme";
import KustomizeOverlay from "./KustomizeOverlay";

jest.mock("../../shared/DiffEditor");
jest.mock("./AceEditorHOC");

const mockKustomizeStep = {
    kustomize: {
        basePath: "base",
        tree: {
            children: [
                {
                    children: [{
                        name: "deployment.yaml",
                        path: "/deployment.yaml",
                    }],
                    name: "/",
                    path: "/",
                },
                {
                    children: [{
                        name: "deployment.yaml",
                        path: "/deployment.yaml",
                    }],
                    name: "overlays",
                    path: "/",
                },
            ],
        }
    },
};

const mockDataLoading = {
    fileContentLoading: false,
};

const mockPatch = "This is a mock patch";

describe("KustomizeOverlay", () => {
    describe("select a file with an existing overlay", () => {
        describe("on click of Show Diff", () => {
            const wrapper = mount(
                <KustomizeOverlay
                  currentStep={mockKustomizeStep}
                  dataLoading={mockDataLoading}
                  modified={""}
                />
            );
            it("makes a request to generate a new modified", () => {
                const spy = jest.spyOn(wrapper.instance(), "handleApplyPatch");

                wrapper.setProps({
                  patch: mockPatch,
                });
                wrapper.setState({
                  selectedFile: "/deployment.yaml",
                });

                const diffToggle = wrapper.find(".diff-toggle");
                diffToggle.simulate("click");

                expect(spy).toHaveBeenCalled();
            });
        });
    });
});
