import React from "react";
import { mount } from "enzyme";
import { MemoryRouter } from "react-router-dom";
import { NavBar } from "./NavBar";

const mockRouterProps = {
  location: {
    pathname: "",
  },
}

const initProps = {
  shipAppMetadata: {
    name: "",
    icon: "",
  },
  channelDetails: {
    name: "",
    icon: "",
  },
  routes: [],
};

describe("NavBar", () => {
  beforeAll(() => {
    // Mocking Image.prototype.src to call onload immediately
    Object.defineProperty(global.Image.prototype, "src", {
      set() {
        this.onload()
      },
    });
  });

  describe("provided shipAppMetadata", () => {
    const wrapper = mount(
      <MemoryRouter initialEntries={["/"]} initialIndex={0}>
        <NavBar
          {...mockRouterProps}
          {...initProps}
        />
      </MemoryRouter>
    );
      it("sets navDetails via shipAppMetadata", async () => {
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
        await wrapper.update();
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
          {...initProps}
        />
      </MemoryRouter>
    );
      it("sets navDetails via channelDetails", async () => {
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
        await wrapper.update();
        const navBar = wrapper.find(NavBar).instance();
        const navDetails = navBar.state.navDetails;
        expect(navDetails.name).toEqual("testChannelDetails");
        expect(navDetails.icon).toEqual("testChannelDetailsIcon");
      });
  });
});
