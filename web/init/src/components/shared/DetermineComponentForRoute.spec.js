import React from "react";
import { shallow } from "enzyme";
import { DetermineComponentForRoute, __RewireAPI__ as RewireAPI } from "./DetermineComponentForRoute";

jest.mock("./DiffEditor");
jest.mock("../kustomize/kustomize_overlay/AceEditorHOC");

const mockPollContentForStep = jest.fn();
const mockKustomizeIntroProps = {
  phase: "kustomize-intro",
  dataLoading: {},
  actions: ["mockKustomizeIntroAction"],
  getContentForStep: jest.fn(),
  routes: [{
    phase: "kustomize",
    id: "mockKustomizeId",
  }],
  finalizeStep: jest.fn(),
  pollContentForStep: mockPollContentForStep,
};

describe("DetermineComponentForRoute", () => {
  describe("skipKustomize", () => {
    it("calls the kustomizeIntro action, kustomize action, and then polls for the kustomize step", async() => {
      const mockHandleAction = jest.fn();

      const mockFetchContentForStep = jest.fn();
      mockFetchContentForStep.mockImplementation(() => ({
        actions: ["mockKustomizeAction"],
      }));
      RewireAPI.__set__("fetchContentForStep", mockFetchContentForStep);

      const wrapper = shallow(<DetermineComponentForRoute {...mockKustomizeIntroProps} />);
      wrapper.instance().handleAction = mockHandleAction;
      wrapper.update();

      await wrapper.instance().skipKustomize();

      expect(mockHandleAction.mock.calls).toHaveLength(2);
      expect(mockHandleAction.mock.calls[0][0]).toEqual("mockKustomizeIntroAction");
      expect(mockHandleAction.mock.calls[1][0]).toEqual("mockKustomizeAction");

      expect(mockPollContentForStep).toHaveBeenCalled();
      expect(mockPollContentForStep.mock.calls[0][0]).toEqual("mockKustomizeId");
    });
  });
});
