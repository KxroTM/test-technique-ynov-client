(function () {
    "use strict";

    const board = document.querySelector("[data-board]");
    if (!board) {
        return;
    }

    let dragged = null;
    let placeholder = null;

    const order = [...board.querySelectorAll(".column")].map(function (column) {
        return column.dataset.status;
    });

    const labels = {};
    board.querySelectorAll(".column").forEach(function (column) {
        labels[column.dataset.status] = column.querySelector(".column-title").textContent.trim();
    });

    function createPlaceholder() {
        const element = document.createElement("div");
        element.className = "drop-placeholder";
        return element;
    }

    function cardAfterPoint(zone, y) {
        const cards = [...zone.querySelectorAll(".note:not(.is-dragging)")];

        return cards.reduce(
            (closest, card) => {
                const box = card.getBoundingClientRect();
                const offset = y - box.top - box.height / 2;
                if (offset < 0 && offset > closest.offset) {
                    return { offset: offset, element: card };
                }
                return closest;
            },
            { offset: Number.NEGATIVE_INFINITY, element: null }
        ).element;
    }

    function refreshColumn(column) {
        const zone = column.querySelector("[data-dropzone]");
        const count = zone.querySelectorAll(".note").length;

        const counter = column.querySelector("[data-count]");
        if (counter) {
            counter.textContent = String(count);
        }

        const empty = zone.querySelector("[data-empty]");
        if (count === 0 && !empty) {
            const message = document.createElement("p");
            message.className = "column-empty";
            message.setAttribute("data-empty", "");
            message.textContent = "Aucune note";
            zone.appendChild(message);
        } else if (count > 0 && empty) {
            empty.remove();
        }
    }

    function refreshProgress() {
        const bar = document.querySelector("[data-progress-bar]");
        if (!bar) {
            return;
        }

        const counts = {};
        let total = 0;

        board.querySelectorAll(".column").forEach(function (column) {
            const size = column.querySelectorAll(".note").length;
            counts[column.dataset.status] = size;
            total += size;
        });

        const share = function (status) {
            return total === 0 ? 0 : Math.floor((counts[status] || 0) * 100 / total);
        };

        bar.querySelectorAll("[data-part]").forEach(function (part) {
            part.style.width = share(part.dataset.part) + "%";
        });

        const donePercent = share("done");
        bar.setAttribute("aria-label", donePercent + " % des notes de cet espace sont terminées");

        const readout = document.querySelector("[data-done-percent]");
        if (readout) {
            readout.textContent = String(donePercent);
        }
    }

    function refreshAllColumns() {
        board.querySelectorAll(".column").forEach(refreshColumn);
        refreshProgress();
    }

    function syncCardControls(card, status) {
        const index = order.indexOf(status);

        card.querySelectorAll("[data-direction]").forEach(function (button) {
            const target = button.dataset.direction === "prev" ? order[index - 1] : order[index + 1];

            if (target) {
                button.value = target;
                button.title = "Déplacer vers « " + labels[target] + " »";
                button.hidden = false;
            } else {
                button.hidden = true;
            }
        });
    }

    function showError(message) {
        const container = board.parentNode;
        const existing = container.querySelector("[data-board-error]");
        if (existing) {
            existing.remove();
        }

        const alert = document.createElement("p");
        alert.className = "alert alert-error";
        alert.setAttribute("role", "alert");
        alert.setAttribute("data-board-error", "");
        alert.textContent = message;

        container.insertBefore(alert, board);
    }

    async function persistStatus(card, status, restore) {
        card.classList.add("is-pending");

        try {
            const response = await fetch("/notes/" + card.dataset.noteId + "/status", {
                method: "POST",
                headers: {
                    "Content-Type": "application/x-www-form-urlencoded",
                    "X-Board-Request": "1"
                },
                body: "status=" + encodeURIComponent(status)
            });

            if (!response.ok) {
                throw new Error("réponse " + response.status);
            }

            card.dataset.status = status;
            syncCardControls(card, status);
        } catch (error) {
            restore();
            showError("Le déplacement n'a pas pu être enregistré. La note a été remise à sa place.");
        } finally {
            card.classList.remove("is-pending");
            refreshAllColumns();
        }
    }

    board.addEventListener("dragstart", function (event) {
        const card = event.target.closest(".note");
        if (!card) {
            return;
        }

        dragged = card;
        placeholder = createPlaceholder();
        card.classList.add("is-dragging");
        event.dataTransfer.effectAllowed = "move";
        event.dataTransfer.setData("text/plain", card.dataset.noteId);
    });

    board.addEventListener("dragend", function () {
        if (dragged) {
            dragged.classList.remove("is-dragging");
        }
        if (placeholder) {
            placeholder.remove();
        }
        board.querySelectorAll(".column").forEach(function (column) {
            column.classList.remove("is-drop-target");
        });
        dragged = null;
        placeholder = null;
    });

    board.addEventListener("dragover", function (event) {
        if (!dragged) {
            return;
        }

        const zone = event.target.closest("[data-dropzone]");
        if (!zone) {
            return;
        }

        event.preventDefault();
        event.dataTransfer.dropEffect = "move";

        board.querySelectorAll(".column").forEach(function (column) {
            column.classList.toggle("is-drop-target", column.contains(zone));
        });

        const reference = cardAfterPoint(zone, event.clientY);
        if (reference) {
            zone.insertBefore(placeholder, reference);
        } else {
            zone.appendChild(placeholder);
        }
    });

    board.addEventListener("drop", function (event) {
        if (!dragged || !placeholder || !placeholder.parentNode) {
            return;
        }

        event.preventDefault();

        const card = dragged;
        const zone = placeholder.parentNode;
        const column = zone.closest(".column");
        const status = column.dataset.status;
        const previousZone = card.parentNode;
        const previousSibling = card.nextElementSibling;
        const previousStatus = card.dataset.status;

        zone.insertBefore(card, placeholder);
        placeholder.remove();
        refreshAllColumns();

        if (status === previousStatus) {
            return;
        }

        persistStatus(card, status, function () {
            if (previousSibling) {
                previousZone.insertBefore(card, previousSibling);
            } else {
                previousZone.appendChild(card);
            }
            card.dataset.status = previousStatus;
        });
    });
})();
