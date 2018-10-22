const buildRepeatKeyString = (key, length) => Array.from({ length }).fill(key).join("");

describe("Ship Init test-charts/modify-chart", () => {
  before(() => {
    cy.visit(Cypress.env("HOST"));
  })

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
        cy.get(".secondary").click();
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
      cy.get(".btn").click();
      cy.location("pathname").should("eq", "/kustomize");
    });
  });
  context("kustomize", () => {
    context("valid line clicked in editor", () => {
      it("generates a stubbed overlay", () => {
        cy.get(".u-marginLeft--normal > :nth-child(1)").click();
        cy.get(".file-contents-wrapper > #brace-editor > .ace_scroller > .ace_content > .ace_text-layer > :nth-child(13)").as("replicaKey");
        cy.get("@replicaKey").trigger("mousemove", { force: true });
        cy.get("@replicaKey").click({ force: true });
      });

      it("allows the stubbed overlay to be edited", () => {
        const backspacesToRemoveToBeModified = buildRepeatKeyString("{backspace}", 14);
        cy.get(".ace_text-input").last().type(
          `${backspacesToRemoveToBeModified}10`,
          { force: true }
        )
      });
      context("valid Kustomize overlay written", () => {
        it("allows the overlay to be saved", () => {
          cy.get(".primary").click();
        });

        it("allows navigation to the overlay finalization step", () => {
          cy.get(".secondary").click();
          cy.location("pathname", {timeout: 5000}).should("eq", "/outro");
        })
      })
    })
  });

  context("outro", () => {
    it("allows the user to complete wizard", () => {
      cy.get(".btn").first().click();
    })
  })
});
