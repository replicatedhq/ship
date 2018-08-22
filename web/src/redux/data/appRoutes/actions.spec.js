import thunk from "redux-thunk";
import configureMockStore from "redux-mock-store";
import { pollContentForStep, __RewireAPI__ as RewireAPI } from "./actions";

jest.useFakeTimers();

describe("appRoutes actions", () => {
  describe("pollContentForStep", () => {
    it("should poll until receiving status of success and invoke the provided cb", (done) => {
      const middlewares = [thunk];
      const mockStore = configureMockStore(middlewares);
      const store = mockStore({
        polling: false,
      });

      const mockFetchContentForStep = jest.fn();
      mockFetchContentForStep.mockImplementation(() => Promise.resolve({
        progress: {
          detail: JSON.stringify({
            status: "success",
          }),
        },
      }));
      RewireAPI.__set__("fetchContentForStep", mockFetchContentForStep);

      const mockCb = jest.fn().mockImplementation(() => done());
      expect(mockCb).not.toBeCalled();

      store.dispatch(pollContentForStep("someRandomId", mockCb));
      const expectedActions = [ { payload: true, type: "POLLING" } ];
      const actions = store.getActions();
      expect(actions).toEqual(expectedActions);

      jest.runOnlyPendingTimers();
      expect(mockFetchContentForStep).toBeCalled();
      expect(mockFetchContentForStep).toHaveBeenCalledTimes(1);
    });
  });
});
