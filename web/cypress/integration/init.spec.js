const buildRepeatKeyString = (key, length) => Array.from({ length }).fill(key).join("");

describe("Ship Init Sourcegraph", () => {
    before(() => {
        cy.visit(Cypress.env("HOST"));
    })
    it("allows navigation to the Helm step", () => {
        cy.get(".btn").click();
    });
    context("required Helm values not entered", () => {
        it("fails with an error message", () => {
            cy.get('.actions-wrapper > .flex > .btn').click();
            cy.get(".actions-wrapper p.u-color--chestnut").should("contain", "[ERROR]")
        });
    });
    context("required Helm values entered", () => {
        it("allows user to proceed to next step", () => {
            // 258 down arrows
            // 3 right arrows
            // 2 backspaces
            const downArrowsToRequiredHelmValue = buildRepeatKeyString("{downarrow}", 258);
            const rightArrowsToRequiredHelmValue = buildRepeatKeyString("{rightArrow}", 3);
            const backspacesToUncommentRequiredHelmValue = buildRepeatKeyString("{backspace}", 2);
            cy.get(".ace_text-input").first().type(
                `${downArrowsToRequiredHelmValue}${rightArrowsToRequiredHelmValue}${backspacesToUncommentRequiredHelmValue}`,
                { force: true, delay: 0 }
            )
            cy.get('.actions-wrapper > .flex > .btn').click()
            cy.get('.flex-auto.flex-column > .flex > .btn').click()
        });
    });
});
