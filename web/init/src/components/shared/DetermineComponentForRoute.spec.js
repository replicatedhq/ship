import React from "react";
import { shallow } from "enzyme";
import { DetermineComponentForRoute, __RewireAPI__ as RewireAPI } from "./DetermineComponentForRoute";

jest.mock("./DiffEditor");
jest.mock("../kustomize/kustomize_overlay/AceEditorHOC");

const mockPollContentForStep = jest.fn();
const mockKustomizeIntroProps = {
  apiEndpoint: "https://ship-api.com",
  phase: "kustomize-intro",
  dataLoading: {},
  actions: ["mockKustomizeIntroAction"],
  getContentForStep: jest.fn(),
  routes: [
    {
      phase: "kustomize",
      id: "mockKustomizeId",
    },
    {
      phase: "kustomize-intro",
      id: "mockKustomizeIntroId",
    },
  ],
  finalizeStep: jest.fn(),
  pollContentForStep: mockPollContentForStep,
  history: {
    goBack: jest.fn()
  },
  currentRoute: {
    id: "mockKustomizeIntroId",
  },
};

describe("DetermineComponentForRoute", () => {
  describe("skipKustomize", () => {
    it("completes kustomize step and polls for the outro", async() => {
      const mockHandleAction = jest.fn();

      const mockFetchContentForStep = jest.fn();
      mockFetchContentForStep.mockImplementation(() => ({
        actions: ["mockKustomizeAction"],
      }));
      RewireAPI.__set__("fetchContentForStep", mockFetchContentForStep);

      const wrapper = shallow(
        <DetermineComponentForRoute
          {...mockKustomizeIntroProps}
          fetchContentForStep={mockFetchContentForStep}
        />
      );
      wrapper.instance().handleAction = mockHandleAction;
      wrapper.update();

      await wrapper.instance().skipKustomize();

      expect(mockFetchContentForStep.mock.calls).toHaveLength(1);
      expect(mockFetchContentForStep.mock.calls[0]).toEqual(["https://ship-api.com", "kustomize"]);

      expect(mockPollContentForStep).toHaveBeenCalled();
      expect(mockPollContentForStep.mock.calls[0][0]).toEqual("kustomize");
    });
  });
});
