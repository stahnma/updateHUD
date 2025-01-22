document.addEventListener("DOMContentLoaded", () => {
    const systemsTable = document.getElementById("systems");
    let sortOrder = {
        column: "hostname", // Default sort column
        ascending: true, // Default sort order
    };

    // Fetch and render systems list
    function fetchSystems() {
        fetch("/api/systems")
            .then((response) => {
                if (!response.ok) {
                    throw new Error(`Failed to fetch systems: ${response.status}`);
                }
                return response.json();
            })
            .then((systems) => {
                renderSystems(systems);
            })
            .catch((error) => {
                console.error("[ERROR] Failed to fetch systems:", error);
            });
    }

    // Render systems data into the table
    function renderSystems(systems) {
        // Sort the systems based on the current sort order
        systems.sort((a, b) => {
            const aValue = a[sortOrder.column] || "";
            const bValue = b[sortOrder.column] || "";
            if (sortOrder.ascending) {
                return aValue.toString().localeCompare(bValue.toString());
            } else {
                return bValue.toString().localeCompare(aValue.toString());
            }
        });

        // Generate table rows
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
                        <td>${system.last_seen}</td>
                    </tr>
                    <tr class="details-row" data-hostname="${system.hostname}" style="display: none;">
                        <td colspan="7">
                            <div class="details-content">Loading...</div>
                        </td>
                    </tr>
                `
            )
            .join("");
        systemsTable.innerHTML = rows;
    }

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

    // Add event listeners to table headers for sorting
    document.querySelectorAll("th.sortable").forEach((header) => {
        header.addEventListener("click", () => {
            const column = header.dataset.column;

            // Toggle sort order if the same column is clicked
            if (sortOrder.column === column) {
                sortOrder.ascending = !sortOrder.ascending;
            } else {
                sortOrder.column = column;
                sortOrder.ascending = true;
            }

            // Fetch and re-render the systems with updated sorting
            fetchSystems();
        });
    });

    // Initial fetch
    fetchSystems();
});

