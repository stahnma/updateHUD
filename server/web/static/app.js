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
    let expandedSystems = new Set(); // Track manually expanded systems

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

    // Get OS icon based on OS name
    function getOSIcon(os) {
        if (!os) return '';
        
        const osLower = os.toLowerCase();
        
        // macOS
        if (osLower.includes('darwin') || osLower.includes('macos') || osLower.includes('mac os')) {
            return '<svg class="os-icon" viewBox="0 0 24 24" width="20" height="20"><path fill="#000" d="M18.71 19.5c-.83 1.24-1.71 2.45-3.05 2.47-1.34.03-1.77-.79-3.29-.79-1.53 0-2 .77-3.27.82-1.31.05-2.3-1.32-3.14-2.53C4.25 17 2.94 12.45 4.7 9.39c.87-1.52 2.43-2.48 4.12-2.51 1.28-.02 2.5.87 3.29.87.78 0 2.26-1.07 3.81-.91.65.03 2.47.26 3.64 1.98-.09.06-2.17 1.28-2.15 3.81.03 3.02 2.65 4.03 2.68 4.04-.03.07-.42 1.44-1.38 2.83M13 3.5c.73-.83 1.94-1.46 2.94-1.5.13 1.17-.34 2.35-1.04 3.19-.69.85-1.83 1.51-2.95 1.42-.15-1.15.41-2.35 1.05-3.11z"/></svg>';
        }
        
        // Debian - red swirl logo
        if (osLower.includes('debian')) {
            return '<svg class="os-icon" viewBox="0 0 24 24" width="20" height="20"><circle fill="#A80030" cx="12" cy="12" r="12"/><path fill="#fff" d="M12 4c4.4 0 8 3.6 8 8s-3.6 8-8 8-8-3.6-8-8 3.6-8 8-8zm0 1.5c-3.6 0-6.5 2.9-6.5 6.5s2.9 6.5 6.5 6.5 6.5-2.9 6.5-6.5S15.6 5.5 12 5.5zm0 1c3 0 5.5 2.5 5.5 5.5S15 17.5 12 17.5 6.5 15 6.5 12 9 6.5 12 6.5z"/></svg>';
        }
        
        // Ubuntu - orange circle with white inner circles
        if (osLower.includes('ubuntu')) {
            return '<svg class="os-icon" viewBox="0 0 24 24" width="20" height="20"><circle fill="#E95420" cx="12" cy="12" r="12"/><circle fill="#fff" cx="12" cy="12" r="7.5" opacity="0.9"/><circle fill="#E95420" cx="12" cy="12" r="4.5"/></svg>';
        }
        
        // SUSE / openSUSE - green gecko/chameleon
        if (osLower.includes('suse') || osLower.includes('opensuse')) {
            return '<svg class="os-icon" viewBox="0 0 24 24" width="20" height="20"><circle fill="#73BA25" cx="12" cy="12" r="12"/><path fill="#fff" d="M12 5c3.9 0 7 3.1 7 7s-3.1 7-7 7-7-3.1-7-7 3.1-7 7-7zm0 1.5c-3 0-5.5 2.5-5.5 5.5S9 17.5 12 17.5 17.5 15 17.5 12 15 6.5 12 6.5zm0 1c2.5 0 4.5 2 4.5 4.5S14.5 16.5 12 16.5 7.5 14.5 7.5 12 9.5 7.5 12 7.5z"/></svg>';
        }
        
        // Red Hat / RHEL - red hat shadowman
        if (osLower.includes('red hat') || osLower.includes('rhel') || osLower.includes('redhat')) {
            return '<svg class="os-icon" viewBox="0 0 24 24" width="20" height="20"><circle fill="#EE0000" cx="12" cy="12" r="12"/><path fill="#fff" d="M12 5c3.9 0 7 3.1 7 7s-3.1 7-7 7-7-3.1-7-7 3.1-7 7-7zm0 1.5c-3 0-5.5 2.5-5.5 5.5S9 17.5 12 17.5 17.5 15 17.5 12 15 6.5 12 6.5zm0 1c2.5 0 4.5 2 4.5 4.5S14.5 16.5 12 16.5 7.5 14.5 7.5 12 9.5 7.5 12 7.5z"/></svg>';
        }
        
        // Rocky Linux - blue-green mountain
        if (osLower.includes('rocky')) {
            return '<svg class="os-icon" viewBox="0 0 24 24" width="20" height="20"><circle fill="#10B981" cx="12" cy="12" r="12"/><path fill="#fff" d="M12 5c3.9 0 7 3.1 7 7s-3.1 7-7 7-7-3.1-7-7 3.1-7 7-7zm0 1.5c-3 0-5.5 2.5-5.5 5.5S9 17.5 12 17.5 17.5 15 17.5 12 15 6.5 12 6.5zm0 1c2.5 0 4.5 2 4.5 4.5S14.5 16.5 12 16.5 7.5 14.5 7.5 12 9.5 7.5 12 7.5z"/></svg>';
        }
        
        // Generic Linux fallback - Tux penguin simplified
        if (osLower.includes('linux')) {
            return '<svg class="os-icon" viewBox="0 0 24 24" width="20" height="20"><circle fill="#000" cx="12" cy="12" r="12"/><circle fill="#fff" cx="12" cy="12" r="9"/><circle fill="#000" cx="10" cy="10" r="1.5"/><circle fill="#000" cx="14" cy="10" r="1.5"/><ellipse fill="#000" cx="12" cy="13" rx="2" ry="1.5"/></svg>';
        }
        
        // Default/unknown OS
        return '';
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
                        <td class="os-cell">${getOSIcon(system.os)} <span class="os-text">${system.os} ${system.os_version || ''}</span></td>
                        <td>${system.architecture}</td>
                        <td>${system.ip}</td>
                        <td>${system.updates_available ?
                            `<span class="update-badge update-available${system.pending_updates ? ' priority-' + getUpdatePriority(system.pending_updates) : ''}"
                                   title="Updates available${system.pending_updates ? ': ' + system.pending_updates.map(u => u.name).join(', ') : ' - Click for details'}">
                                Updates${system.pending_updates ? ` (${system.pending_updates.length})` : ' (click for details)'}
                                ${system.pending_updates && getUpdatePriority(system.pending_updates) === 'high' ? ' ⚠️' : ''}
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

        // Restore expanded rows for systems that were manually expanded
        expandedSystems.forEach(hostname => {
            const detailsRow = document.querySelector(`.details-row[data-hostname="${hostname}"]`);
            const chevron = document.querySelector(`tr[data-hostname="${hostname}"] .chevron`);
            if (detailsRow && chevron) {
                detailsRow.style.display = "table-row";
                chevron.textContent = "▼";
                // Load details if not already loaded
                const detailsContent = detailsRow.querySelector(".details-content");
                if (!detailsContent.dataset.loaded || Date.now() - detailsContent.dataset.loadedTime > 60000) {
                    loadSystemDetails(hostname, detailsContent);
                }
            }
        });
    }

    // Function to load system details
    function loadSystemDetails(hostname, detailsContent) {
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
            expandedSystems.add(hostname); // Track as manually expanded

            // Fetch details only if not already loaded or if data is stale
            if (!detailsContent.dataset.loaded || Date.now() - detailsContent.dataset.loadedTime > 60000) {
                loadSystemDetails(hostname, detailsContent);
            }
        } else {
            detailsRow.style.display = "none";
            chevron.textContent = "▶";
            expandedSystems.delete(hostname); // Remove from expanded set
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

    // Expand all systems
    function expandAll() {
        document.querySelectorAll('.details-row').forEach(detailsRow => {
            const hostname = detailsRow.dataset.hostname;
            const chevron = document.querySelector(`tr[data-hostname="${hostname}"] .chevron`);
            if (chevron && detailsRow.style.display === "none") {
                detailsRow.style.display = "table-row";
                chevron.textContent = "▼";
                expandedSystems.add(hostname);
                const detailsContent = detailsRow.querySelector(".details-content");
                if (!detailsContent.dataset.loaded || Date.now() - detailsContent.dataset.loadedTime > 60000) {
                    loadSystemDetails(hostname, detailsContent);
                }
            }
        });
    }

    // Collapse all systems
    function collapseAll() {
        document.querySelectorAll('.details-row').forEach(detailsRow => {
            const hostname = detailsRow.dataset.hostname;
            const chevron = document.querySelector(`tr[data-hostname="${hostname}"] .chevron`);
            if (chevron && detailsRow.style.display !== "none") {
                detailsRow.style.display = "none";
                chevron.textContent = "▶";
                expandedSystems.delete(hostname);
            }
        });
    }

    // Add event listeners for expand/collapse all buttons
    const expandAllBtn = document.getElementById("expand-all-btn");
    const collapseAllBtn = document.getElementById("collapse-all-btn");
    
    if (expandAllBtn) {
        expandAllBtn.addEventListener("click", expandAll);
    }
    
    if (collapseAllBtn) {
        collapseAllBtn.addEventListener("click", collapseAll);
    }

    // Initial fetch and WebSocket connection
    fetchSystems();
    initWebSocket();
});

