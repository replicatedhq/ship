import React from "react";
import { shallow } from "enzyme";
import HelmValuesEditor from "./HelmValuesEditor";

const mockShipAppMetadata = {
  values: "some: values",
  readme: "a readme",
  name: "test",
}

const mockGetStep = {};

describe("HelmValuesEditor", () => {
  describe("with invalid helm values", () => {
    describe("saved and finalized", () => {
      it("allows for the save button to be clicked after an error", async() => {
        const mockSaveValues = jest.fn();
        mockSaveValues.mockImplementation(() => Promise.resolve({ errors: ["Test error"]}));

        const mockSpecValue = "some: different values";
        const wrapper = shallow(
          <HelmValuesEditor
            shipAppMetadata={mockShipAppMetadata}
            saveValues={mockSaveValues}
            getStep={mockGetStep}
          />
        );

        wrapper.setState({ specValue: mockSpecValue });
        await wrapper.instance().handleSaveValues(true);

        expect(mockSaveValues.mock.calls).toHaveLength(1);
        const elements = wrapper.find("button");
        expect(elements).toHaveLength(2);
        elements.forEach((element) => {
          expect(element.prop("disabled")).toEqual(false);
        });
      });
    });
    describe("saved", () => {
      it("allows for the save button to be clicked after an error", async() => {
        const mockSaveValues = jest.fn();
        mockSaveValues.mockImplementation(() => Promise.resolve({ errors: ["Test error"]}));

        const mockSpecValue = "some: different values";
        const wrapper = shallow(
          <HelmValuesEditor
            shipAppMetadata={mockShipAppMetadata}
            saveValues={mockSaveValues}
            getStep={mockGetStep}
          />
        );

        wrapper.setState({ specValue: mockSpecValue });
        await wrapper.instance().handleSaveValues(false);

        expect(mockSaveValues.mock.calls).toHaveLength(1);
        const elements = wrapper.find("button");
        expect(elements).toHaveLength(2);
        elements.forEach((element) => {
          expect(element.prop("disabled")).toEqual(false);
        });
      });
    });
  });
});
