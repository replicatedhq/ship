import React from "react";

import StepTerraform from "./StepTerraform";
import renderer from "react-test-renderer";

const mockProps = {
  location: {
    pathname: "",
  },
  routeId: "",
  startPoll: jest.fn(),
  gotoRoute: jest.fn(),
  initializeStep: jest.fn(),
};

describe("StepTerraform", () => {
  describe("provided a status of message", () => {
    it("renders step message", () => {
      const propsWithMessage = {
        ...mockProps,
        status: {
          type: "json",
          detail: JSON.stringify({
            status: "message",
            message: {
              contents: "this is the message contents",
            },
          }),
        },
      };
      const tree = renderer.create(
        <StepTerraform
          {...propsWithMessage}
        />
      ).toJSON();
      expect(tree).toMatchSnapshot();
    });
  });
  describe("provided a status of error", () => {
    it("renders the error message", () => {
      const propsWithError = {
        ...mockProps,
        status: {
          type: "json",
          detail: JSON.stringify({
            status: "error",
            message: "this is an error",
          }),
        }
      };
      const tree = renderer.create(
        <StepTerraform
        {...propsWithError}
        />
      ).toJSON();
      expect(tree).toMatchSnapshot();
    });
  });
  describe("provided a status of working", () => {
    it("renders a loader", () => {
      const propsWithError = {
        ...mockProps,
        status: {
          type: "json",
          detail: JSON.stringify({
            status: "working",
          }),
        }
      };
      const tree = renderer.create(
        <StepTerraform
        {...propsWithError}
        />
      ).toJSON();
      expect(tree).toMatchSnapshot();
    });
  });
});
