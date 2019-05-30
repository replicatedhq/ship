const buildRepeatKeyString = (key, length) => Array.from({ length }).fill(key).join("");
const SECONDS = 1000;
describe("Ship Init test-charts/modify-chart", () => {
  before(() => {
    cy.visit(Cypress.env("HOST"));
  })

  context("intro", () => {
    it("allows navigation to the Helm step", () => {
      cy.get(".btn").click();
      cy.location("pathname").should("eq", "/values")
    });
  });

  context("values", () => {
    context("required Helm values entered", () => {
      it("successfully saves Helm values", () => {
        const downArrowsToRequiredHelmValue = buildRepeatKeyString("{downarrow}", 16);
        const rightArrowsToRequiredHelmValue = buildRepeatKeyString("{rightArrow}", 16);
        const backspacesToDeleteValue = buildRepeatKeyString("{backspace}", 5);
        cy.get(".ace_text-input").first().type(
          `${downArrowsToRequiredHelmValue}${rightArrowsToRequiredHelmValue}${backspacesToDeleteValue}true`,
          { force: true, delay: 0 }
        );
        cy.get(".primary").click();
        cy.get(".u-color--vidaLoca").contains("Values saved");
      });

      it("allows navigation to the render step", () => {
        cy.get(".secondary.hv-save").click();
        cy.location("pathname").should("eq", "/render")
      })
    });
  });

  context("render", () => {
    it("allows navigation to the kustomize-intro step", () => {
      cy.location("pathname").should("eq", "/kustomize-intro");
    });
  });

  context("kustomize-intro", () => {
    it("allows navigation to the kustomize step", () => {
      cy.get(".renderActions-wrapper .btn.primary").click();
      cy.location("pathname").should("eq", "/kustomize");
    });
  });
  context("kustomize", () => {
    context("valid line clicked in editor", () => {
      it("generates a stubbed overlay", () => {
        cy.get(".FileTree-wrapper > ul > li").first().click();
        cy.get(".file-contents-wrapper > #brace-editor > .ace_scroller > .ace_content > .ace_text-layer > :nth-child(1)").as("replicaKey");
        cy.get("@replicaKey").trigger("mousemove", { force: true });
        cy.get("@replicaKey").click({ force: true });
      });

      it("allows the stubbed overlay to be edited", () => {
        cy.get(".ace_text-input").last().type(
          `{end}{backspace}2`,
          { force: true }
        )
      });
      context("valid Kustomize overlay written", () => {
        it("allows the overlay to be saved", () => {
          cy.get(".save-btn").click();
        });

        it("allows navigation to the overlay finalization step", () => {
          cy.get(".finalize-btn").click();
          cy.wait(10.5 * SECONDS);
          cy.location("pathname", {timeout: 10000}).should("eq", "/outro");
        })
      })
    })
  });

  context("outro", () => {
    it("allows the user to complete wizard", () => {
      cy.get(".btn.primary", {timeout: 10000}).first().click({force: true});
    })
  })
});
