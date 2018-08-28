const buildRepeatKeyString = (key, length) => Array.from({ length }).fill(key).join("");

describe("Ship Init Sourcegraph", () => {
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
    context("required Helm values not entered", () => {
      it("fails with an error message", () => {
        cy.get(".actions-wrapper > .flex > .btn").click();
        cy.get(".actions-wrapper p.u-color--chestnut").should("contain", "[ERROR]")
      });
    });
    context("required Helm values entered", () => {
      it("successfully saves Helm values", () => {
        const downArrowsToRequiredHelmValue = buildRepeatKeyString("{downarrow}", 258);
        const rightArrowsToRequiredHelmValue = buildRepeatKeyString("{rightArrow}", 3);
        const backspacesToUncommentRequiredHelmValue = buildRepeatKeyString("{backspace}", 2);
        cy.get(".ace_text-input").first().type(
          `${downArrowsToRequiredHelmValue}${rightArrowsToRequiredHelmValue}${backspacesToUncommentRequiredHelmValue}`,
          { force: true, delay: 0 }
        )
        cy.get(".actions-wrapper > .flex > .btn").click()
      });

      it("allows navigation to the Kustomize intro step", () => {
        cy.get(".flex-auto.flex-column > .flex > .btn").click();
        cy.location("pathname").should("eq", "/kustomize-intro")
      })
    });
  })
  context("kustomize-intro", () => {
    it("allows navigation to the kustomize step", () => {
      cy.get(".btn").click();
      cy.location("pathname").should("eq", "/kustomize")
    })
  })
  context("kustomize", () => {
    context("valid line clicked in editor", () => {
      it("generates a stubbed overlay", () => {
        cy.visit(Cypress.env("HOST") + "/kustomize");
        cy.get(":nth-child(3) > .u-marginLeft--normal > :nth-child(2)").click()
        cy.get(".file-contents-wrapper > #brace-editor > .ace_scroller > .ace_content > .ace_text-layer > :nth-child(13)").as("replicaKey")
        cy.get("@replicaKey").trigger("mousemove", { force: true })
        cy.get("@replicaKey").click({ force: true })
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
          cy.get(".layout-footer-actions > .flex > .btn").click();
        });

        it("allows navigation to the overlay finalization step", () => {
          cy.get(".flex-auto.flex-column > .flex > .btn").click();
          cy.location("pathname").should("eq", "/outro")
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
