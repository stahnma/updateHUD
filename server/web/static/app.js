// Function to open the modal with the provided content
function openModal(content) {
    const modal = document.getElementById("modal");
    const modalOverlay = document.getElementById("modal-overlay");

    // Set modal content
    modal.innerHTML = content;

    // Show modal and overlay
    modal.style.display = "block";
    modalOverlay.style.display = "block";
}

// Function to close the modal
function closeModal() {
    const modal = document.getElementById("modal");
    const modalOverlay = document.getElementById("modal-overlay");

    // Hide modal and overlay
    modal.style.display = "none";
    modalOverlay.style.display = "none";
}

// Event listener for fetching and displaying system details
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
                        <td>${system.hostname}</td>
                        <td>${system.os} ${system.os_version}</td>
                        <td>${system.architecture}</td>
                        <td>${system.ip}</td>
                        <td>${system.updates_available ? "Yes" : "No"}</td>
                    </tr>
                `
                )
                .join("");
            systemsTable.innerHTML = rows;
        })
        .catch((error) => {
            console.error("[ERROR] Failed to fetch systems:", error);
        });

    // Handle row click to fetch and display system details
    systemsTable.addEventListener("click", (event) => {
        const row = event.target.closest("tr");
        if (!row || !row.dataset.hostname) {
            console.error("No valid row or hostname found for the clicked element.");
            return;
        }

        const hostname = row.dataset.hostname;
        console.log(`[DEBUG] Fetching details for hostname: ${hostname}`);

        fetch(`/api/systems/${hostname}`)
            .then((response) => {
                console.log(`[DEBUG] Response received for hostname: ${hostname}`, response);
                if (!response.ok) {
                    return response.text().then((text) => {
                        console.error(
                            `[ERROR] Server responded with status: ${response.status}. Response body: ${text}`
                        );
                        throw new Error(`HTTP error! Status: ${response.status}`);
                    });
                }
                return response.json();
            })
            .then((data) => {
                console.log(`[DEBUG] Successfully fetched data for hostname: ${hostname}`, data);
                const content = `
                    <h2>System Details</h2>
                    <ul>
                        <li><strong>Hostname:</strong> ${data.hostname}</li>
                        <li><strong>OS:</strong> ${data.os} ${data.os_version}</li>
                        <li><strong>Architecture:</strong> ${data.architecture}</li>
                        <li><strong>IP Address:</strong> ${data.ip}</li>
                        <li><strong>Updates Available:</strong> ${
                            data.updates_available ? "Yes" : "No"
                        }</li>
                    </ul>
                    <h3>Pending Updates</h3>
                    <ul>
                        ${data.pending_updates
                            .map(
                                (update) =>
                                    `<li>${update.name} (${update.source}) - Version: ${update.version}</li>`
                            )
                            .join("")}
                    </ul>
                    <button onclick="closeModal()">Close</button>
                `;
                openModal(content);
            })
            .catch((error) => {
                console.error(
                    `[ERROR] Failed to fetch system details for hostname: ${hostname}:`,
                    error
                );
                openModal(
                    `<p>Failed to load system details. Please try again.</p><pre>${error.message}</pre>`
                );
            });
    });
});

