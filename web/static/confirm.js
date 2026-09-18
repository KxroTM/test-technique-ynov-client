// Remplace la boîte de dialogue du navigateur par une fenêtre modale aux couleurs de l'application.
// Sans ce script, les formulaires marqués data-confirm s'envoient directement.

(function () {
    "use strict";

    const dialog = document.querySelector("[data-confirm-dialog]");
    if (!dialog || typeof dialog.showModal !== "function") {
        return;
    }

    const title = dialog.querySelector("[data-dialog-title]");
    const detail = dialog.querySelector("[data-dialog-detail]");

    let pendingForm = null;

    document.addEventListener("submit", function (event) {
        const form = event.target.closest("form[data-confirm]");
        if (!form) {
            return;
        }

        event.preventDefault();

        pendingForm = form;
        title.textContent = form.dataset.confirm;
        detail.textContent = form.dataset.confirmDetail || "";
        detail.hidden = !form.dataset.confirmDetail;

        dialog.showModal();
    });

    // Un clic sur le fond ferme la fenêtre, comme le ferait la touche Échap
    dialog.addEventListener("click", function (event) {
        if (event.target === dialog) {
            dialog.close("cancel");
        }
    });

    dialog.addEventListener("close", function () {
        const form = pendingForm;
        pendingForm = null;

        if (dialog.returnValue !== "confirm" || !form) {
            return;
        }

        // submit() n'émet pas d'évènement submit, l'interception ne se redéclenche donc pas
        form.submit();
    });
})();
