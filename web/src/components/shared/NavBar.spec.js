import React from "react";
import { mount } from "enzyme";
import { MemoryRouter } from "react-router-dom";
import { NavBar } from "./NavBar";

const mockRouterProps = {
  location: {
    pathname: "",
  },
}

describe("NavBar", () => {
  describe("provided shipAppMetadata", () => {
    const wrapper = mount(
      <MemoryRouter initialEntries={["/"]} initialIndex={0}>
        <NavBar
          {...mockRouterProps}
          shipAppMetadata={{
            name: "",
            icon: "",
          }}
          channelDetails={{
            name: "",
            icon: "",
          }}
        />
      </MemoryRouter>
    );
      it("sets navDetails via shipAppMetadata", () => {
        wrapper.setProps({
          children: React.cloneElement(
            wrapper.props().children,
            {
              ...mockRouterProps,
              shipAppMetadata: {
                name: "testHelm",
                icon: "testHelmIcon",
              },
            },
          ),
        });
        const navBar = wrapper.find(NavBar).instance();
        const navDetails = navBar.state.navDetails;
        expect(navDetails.name).toEqual("testHelm");
        expect(navDetails.icon).toEqual("testHelmIcon");
      });
  });
  describe("provided channelDetails", () => {
    const wrapper = mount(
      <MemoryRouter initialEntries={["/"]} initialIndex={0}>
        <NavBar
          {...mockRouterProps}
          shipAppMetadata={{
            name: "",
            icon: "",
          }}
          channelDetails={{
            name: "",
            icon: "",
          }}
        />
      </MemoryRouter>
    );
      it("sets navDetails via channelDetails", () => {
        wrapper.setProps({
          children: React.cloneElement(
            wrapper.props().children,
            {
              ...mockRouterProps,
              channelDetails: {
                channelName: "testChannelDetails",
                icon: "testChannelDetailsIcon",
              },
            },
          ),
        });
        const navBar = wrapper.find(NavBar).instance();
        const navDetails = navBar.state.navDetails;
        expect(navDetails.name).toEqual("testChannelDetails");
        expect(navDetails.icon).toEqual("testChannelDetailsIcon");
      });
  });
});
