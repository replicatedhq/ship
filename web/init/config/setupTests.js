import Enzyme from "enzyme";
import Adapter from "enzyme-adapter-react-16";
import "jest-enzyme";

global.localStorage = {
  getItem: () => {},
  setItem: () => {},
  clear: () => {}
};

Enzyme.configure({ adapter: new Adapter() });
