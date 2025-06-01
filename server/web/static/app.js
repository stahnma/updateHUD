document.addEventListener("DOMContentLoaded", () => {
    const systemsTable = document.getElementById("systems");
    let sortOrder = {
        column: "hostname", // Default sort column
        ascending: true, // Default sort order
    };
    let systemsData = []; // Store the current systems data
    let ws = null;
    let reconnectAttempts = 0;
    const MAX_RECONNECT_ATTEMPTS = 10;
    const INITIAL_RECONNECT_DELAY = 1000; // 1 second

    // Initialize WebSocket connection with exponential backoff
    function initWebSocket() {
        if (ws !== null && ws.readyState !== WebSocket.CLOSED) {
            console.log("[DEBUG] WebSocket connection already exists");
            return;
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;
        console.log(`[DEBUG] Attempting to connect to WebSocket at ${wsUrl}`);

        try {
            ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                console.log("[INFO] WebSocket connection established");
                reconnectAttempts = 0; // Reset reconnection attempts on successful connection
                
                // Set up ping interval
                const pingInterval = setInterval(() => {
                    if (ws.readyState === WebSocket.OPEN) {
                        ws.send('ping');
                    } else {
                        clearInterval(pingInterval);
                    }
                }, 30000); // Send ping every 30 seconds
            };

            ws.onmessage = (event) => {
                try {
                    const update = JSON.parse(event.data);
                    
                    // Update the systemsData array
                    const index = systemsData.findIndex(s => s.hostname === update.hostname);
                    if (index !== -1) {
                        systemsData[index] = update;
                    } else {
                        systemsData.push(update);
                    }
                    
                    // Re-render the table with the updated data
                    renderSystems(systemsData);
                } catch (error) {
                    console.error("[ERROR] Failed to process WebSocket message:", error);
                }
            };

            ws.onclose = (event) => {
                console.log(`[WARN] WebSocket connection closed. Code: ${event.code}, Reason: ${event.reason}`);
                clearWebSocket();
                
                // Implement exponential backoff for reconnection
                if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
                    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000); // Cap at 30 seconds
                    console.log(`[INFO] Attempting to reconnect in ${delay/1000} seconds... (Attempt ${reconnectAttempts + 1}/${MAX_RECONNECT_ATTEMPTS})`);
                    setTimeout(() => {
                        reconnectAttempts++;
                        initWebSocket();
                    }, delay);
                } else {
                    console.error("[ERROR] Maximum reconnection attempts reached. Please refresh the page.");
                }
            };

            ws.onerror = (error) => {
                console.error("[ERROR] WebSocket error:", error);
            };

        } catch (error) {
            console.error("[ERROR] Failed to create WebSocket connection:", error);
            clearWebSocket();
        }
    }

    // Clean up WebSocket connection
    function clearWebSocket() {
        if (ws !== null) {
            // Remove all event listeners to prevent memory leaks
            ws.onopen = null;
            ws.onclose = null;
            ws.onmessage = null;
            ws.onerror = null;
            
            // Close the connection if it's still open
            if (ws.readyState === WebSocket.OPEN) {
                ws.close();
            }
            ws = null;
        }
    }

    // Handle page visibility changes
    document.addEventListener('visibilitychange', () => {
        if (document.visibilityState === 'visible') {
            // Attempt to reconnect when page becomes visible
            if (ws === null || ws.readyState === WebSocket.CLOSED) {
                console.log("[INFO] Page became visible, attempting to reconnect WebSocket");
                reconnectAttempts = 0; // Reset reconnection attempts
                initWebSocket();
            }
        }
    });

    // Handle page unload
    window.addEventListener('beforeunload', () => {
        clearWebSocket();
    });

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
                systemsData = systems; // Update the stored data
                renderSystems(systems);
            })
            .catch((error) => {
                console.error("[ERROR] Failed to fetch systems:", error);
            });
    }

    // Format timestamp to be more user-friendly
    function formatTimestamp(isoTimestamp) {
        const date = new Date(isoTimestamp);
        const months = ['Jan', 'Feb', 'Mar', 'Apr', 'May', 'Jun', 'Jul', 'Aug', 'Sep', 'Oct', 'Nov', 'Dec'];
        const day = date.getUTCDate().toString().padStart(2, '0');
        const month = months[date.getUTCMonth()];
        const year = date.getUTCFullYear();
        const hours = date.getUTCHours().toString().padStart(2, '0');
        const minutes = date.getUTCMinutes().toString().padStart(2, '0');
        const seconds = date.getUTCSeconds().toString().padStart(2, '0');
        
        return `${day} ${month} ${year} ${hours}:${minutes}:${seconds}`;
    }

    function getUpdatePriority(updates) {
        if (!updates) return 'none';
        // Check for security updates
        const hasSecurityUpdates = updates.some(u =>
            u.name.toLowerCase().includes('security') ||
            u.source.toLowerCase().includes('security')
        );
        if (hasSecurityUpdates) return 'high';
        if (updates.length > 5) return 'medium';
        if (updates.length > 0) return 'low';
        return 'none';
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
                        <td>${system.os} ${system.os_version || ''}</td>
                        <td>${system.architecture}</td>
                        <td>${system.ip}</td>
                        <td>${system.updates_available ?
                            `<span class="update-badge update-available priority-${getUpdatePriority(system.pending_updates)}"
                                   title="Updates available: ${system.pending_updates ? system.pending_updates.map(u => u.name).join(', ') : 'Click for details'}">
                                Updates (${system.pending_updates ? system.pending_updates.length : '?'})
                                ${getUpdatePriority(system.pending_updates) === 'high' ? ' ⚠️' : ''}
                            </span>` :
                            `<span class="update-badge up-to-date">Up to date</span>`
                        }</td>
                        <td>${formatTimestamp(system.last_seen)}</td>
                    </tr>
                    <tr class="details-row" data-hostname="${system.hostname}" style="display: none;">
                        <td colspan="7">
                            <div class="details-content">Loading...</div>
                        </td>
                    </tr>`
            )
            .join("");
        systemsTable.innerHTML = rows;

        // Restore expanded rows
        document.querySelectorAll('.details-row[style="display: table-row"]').forEach(row => {
            const hostname = row.dataset.hostname;
            const chevron = document.querySelector(`tr[data-hostname="${hostname}"] .chevron`);
            if (chevron) {
                chevron.textContent = "▼";
            }
        });
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

            // Fetch details only if not already loaded or if data is stale
            if (!detailsContent.dataset.loaded || Date.now() - detailsContent.dataset.loadedTime > 60000) {
                fetch(`/api/systems/${hostname}`)
                    .then((response) => {
                        if (!response.ok) {
                            throw new Error(`Failed to fetch system details: ${response.status}`);
                        }
                        return response.json();
                    })
                    .then((data) => {
                        if (data.pending_updates && data.pending_updates.length > 0) {
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
                        } else {
                            detailsContent.innerHTML = `
                                <h3>No pending updates for ${data.hostname}</h3>
                            `;
                        }
                        detailsContent.dataset.loaded = "true";
                        detailsContent.dataset.loadedTime = Date.now();
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

            // Re-render the systems with updated sorting
            renderSystems(systemsData);
        });
    });

    // Initial fetch and WebSocket connection
    fetchSystems();
    initWebSocket();
});

