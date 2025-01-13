document.addEventListener("DOMContentLoaded", () => {
    const systemsTable = document.getElementById("systems");

    // Fetch and render systems list
    fetch("/api/systems")
        .then((response) => {
            if (!response.ok) {
                throw new Error(`Failed to fetch systems: ${response.status}`);
            }
            return response.json();
        })
        .then((systems) => {
            const rows = systems
                .map(
                    (system) => `
                    <tr data-hostname="${system.hostname}">
                        <td class="chevron-cell"><span class="chevron">▶</span></td>
                        <td>${system.hostname}</td>
                        <td>${system.os} ${system.os_version}</td>
                        <td>${system.architecture}</td>
                        <td>${system.ip}</td>
                        <td>${system.updates_available ? "Yes" : "No"}</td>
                    </tr>
                    <tr class="details-row" data-hostname="${system.hostname}" style="display: none;">
                        <td colspan="6">
                            <div class="details-content">Loading...</div>
                        </td>
                    </tr>
                `
                )
                .join("");
            systemsTable.innerHTML = rows;
        })
        .catch((error) => {
            console.error("[ERROR] Failed to fetch systems:", error);
        });

    // Handle row toggle for details
    systemsTable.addEventListener("click", (event) => {
        const chevron = event.target.closest(".chevron");
        if (!chevron) return;

        const row = chevron.closest("tr");
        const hostname = row.dataset.hostname;
        const detailsRow = document.querySelector(`.details-row[data-hostname="${hostname}"]`);
        const detailsContent = detailsRow.querySelector(".details-content");

        // Toggle visibility
        if (detailsRow.style.display === "none") {
            detailsRow.style.display = "table-row";
            chevron.textContent = "▼";

            // Fetch details only if not already loaded
            if (!detailsContent.dataset.loaded) {
                fetch(`/api/systems/${hostname}`)
                    .then((response) => {
                        if (!response.ok) {
                            throw new Error(`Failed to fetch system details: ${response.status}`);
                        }
                        return response.json();
                    })
                    .then((data) => {
                        const updatesList = data.pending_updates
                            .map(
                                (update) =>
                                    `<tr>
                                        <td>${update.name}</td>
                                        <td>${update.version || "N/A"}</td>
                                        <td>${update.source}</td>
                                    </tr>`
                            )
                            .join("");
                        detailsContent.innerHTML = `
                            <h3>Pending Updates for ${data.hostname}</h3>
                            <table class="updates-table">
                                <thead>
                                    <tr>
                                        <th>Package</th>
                                        <th>Version</th>
                                        <th>Source</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    ${updatesList}
                                </tbody>
                            </table>
                        `;
                        detailsContent.dataset.loaded = "true";
                    })
                    .catch((error) => {
                        console.error(
                            `[ERROR] Failed to fetch system details for hostname: ${hostname}:`,
                            error
                        );
                        detailsContent.innerHTML = `
                            <p>Error loading details. Please try again.</p>
                        `;
                    });
            }
        } else {
            detailsRow.style.display = "none";
            chevron.textContent = "▶";
        }
    });
});

