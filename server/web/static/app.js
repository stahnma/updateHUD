document.addEventListener("DOMContentLoaded", () => {
    const systemsTable = document.getElementById("systems");
    if (!systemsTable) {
        console.error("[ERROR] Systems table element not found!");
        return;
    }
    let sortOrder = {
        column: "hostname", // Default sort column
        ascending: true, // Default sort order
    };
    let systemsData = []; // Store the current systems data - always initialize as empty array
    let ws = null;
    let pingIntervalId = null; // Store ping interval ID for cleanup
    let reconnectAttempts = 0;
    const MAX_RECONNECT_ATTEMPTS = 10;
    const INITIAL_RECONNECT_DELAY = 1000; // 1 second
    let expandedSystems = new Set(); // Track manually expanded systems

    // Helper function to escape HTML
    function escapeHtml(text) {
        if (text == null) return '';
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Initialize WebSocket connection with exponential backoff
    function initWebSocket() {
        if (ws !== null && ws.readyState !== WebSocket.CLOSED) {
            return;
        }

        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${window.location.host}/ws`;

        try {
            ws = new WebSocket(wsUrl);

            ws.onopen = () => {
                reconnectAttempts = 0; // Reset reconnection attempts on successful connection
                
                // Clear any existing ping interval
                if (pingIntervalId !== null) {
                    clearInterval(pingIntervalId);
                    pingIntervalId = null;
                }
                
                // Set up ping interval
                pingIntervalId = setInterval(() => {
                    if (ws !== null && ws.readyState === WebSocket.OPEN) {
                        ws.send('ping');
                    } else {
                        if (pingIntervalId !== null) {
                            clearInterval(pingIntervalId);
                            pingIntervalId = null;
                        }
                    }
                }, 30000); // Send ping every 30 seconds
            };

            ws.onmessage = (event) => {
                try {
                    // Ensure systemsData is always an array
                    if (!Array.isArray(systemsData)) {
                        systemsData = [];
                    }
                    
                    const update = JSON.parse(event.data);
                    
                    // Validate that update has required fields
                    if (!update || !update.hostname) {
                        console.warn("Invalid WebSocket update received, missing hostname");
                        return;
                    }
                    
                    // Update the systemsData array
                    const index = systemsData.findIndex(s => s && s.hostname === update.hostname);
                    if (index !== -1) {
                        systemsData[index] = update;
                    } else {
                        systemsData.push(update);
                    }
                    
                    // Render the updated systems
                    renderSystems(Array.from(systemsData));
                } catch (error) {
                    console.error("Failed to process WebSocket message:", error);
                }
            };

            ws.onclose = (event) => {
                clearWebSocket();
                
                // Implement exponential backoff for reconnection
                if (reconnectAttempts < MAX_RECONNECT_ATTEMPTS) {
                    const delay = Math.min(1000 * Math.pow(2, reconnectAttempts), 30000); // Cap at 30 seconds
                    setTimeout(() => {
                        reconnectAttempts++;
                        initWebSocket();
                    }, delay);
                } else {
                    console.error("Maximum reconnection attempts reached. Please refresh the page.");
                }
            };

            ws.onerror = (error) => {
                console.error("WebSocket error:", error);
            };

        } catch (error) {
            console.error("[ERROR] Failed to create WebSocket connection:", error);
            clearWebSocket();
        }
    }

    // Clean up WebSocket connection
    function clearWebSocket() {
        // Clear ping interval
        if (pingIntervalId !== null) {
            clearInterval(pingIntervalId);
            pingIntervalId = null;
        }
        
        if (ws !== null) {
            // Remove all event listeners to prevent memory leaks
            ws.onopen = null;
            ws.onclose = null;
            ws.onmessage = null;
            ws.onerror = null;
            
            // Close the connection if it's still open
            if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
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
                // Ensure systemsData is always an array
                systemsData = Array.isArray(systems) ? systems : [];
                renderSystems(systemsData);
            })
            .catch((error) => {
                console.error("Failed to fetch systems:", error);
                // Ensure systemsData is initialized even on error
                if (!Array.isArray(systemsData)) {
                    systemsData = [];
                }
                renderSystems(systemsData);
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
            return '<img class="os-icon" src="/static/images/macos-30.png" alt="macOS" width="20" height="20">';
        }
        
        // Debian
        if (osLower.includes('debian')) {
            return '<img class="os-icon" src="/static/images/debian-48.png" alt="Debian" width="20" height="20">';
        }
        
        // Fedora
        if (osLower.includes('fedora')) {
            return '<img class="os-icon" src="/static/images/fedora-48.png" alt="Fedora" width="20" height="20">';
        }
        
        // Red Hat / RHEL
        if (osLower.includes('red hat') || osLower.includes('rhel') || osLower.includes('redhat')) {
            return '<img class="os-icon" src="/static/images/red-hat-48.png" alt="Red Hat" width="20" height="20">';
        }
        
        // Ubuntu - use generic Linux icon
        if (osLower.includes('ubuntu')) {
            return '<img class="os-icon" src="/static/images/linux-48.png" alt="Ubuntu" width="20" height="20">';
        }
        
        // SUSE / openSUSE - use generic Linux icon
        if (osLower.includes('suse') || osLower.includes('opensuse')) {
            return '<img class="os-icon" src="/static/images/linux-48.png" alt="SUSE" width="20" height="20">';
        }
        
        // Rocky Linux - use generic Linux icon
        if (osLower.includes('rocky')) {
            return '<img class="os-icon" src="/static/images/linux-48.png" alt="Rocky Linux" width="20" height="20">';
        }
        
        // Generic Linux fallback - use Linux icon
        if (osLower.includes('linux') || osLower.includes('armbian')) {
            return '<img class="os-icon" src="/static/images/linux-48.png" alt="Linux" width="20" height="20">';
        }
        
        // Default/unknown OS
        return '';
    }

    // Render systems data into the table
    function renderSystems(systems) {
        // Ensure systemsTable exists
        if (!systemsTable) {
            console.error("Systems table element not found!");
            return;
        }
        
        // Handle null/undefined systems array
        if (!systems || !Array.isArray(systems)) {
            systems = [];
        }

        // Show message if no systems exist yet
        if (systems.length === 0) {
            systemsTable.innerHTML = `
                <tr>
                    <td colspan="7" style="text-align: center; padding: 2rem; color: #666;">
                        No systems have checked in yet. Systems will appear here automatically as they connect.
                    </td>
                </tr>
            `;
            return;
        }
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
        try {
            const rows = systems
                .map(
                    (system) => {
                        if (!system || !system.hostname) {
                            return '';
                        }
                        return `
                    <tr data-hostname="${system.hostname}">
                        <td class="chevron-cell"><span class="chevron">▶</span></td>
                        <td>${escapeHtml(system.hostname)}</td>
                        <td class="os-cell">${getOSIcon(system.os)} <span class="os-text">${escapeHtml(system.os || '')} ${escapeHtml(system.os_version || '')}</span></td>
                        <td>${escapeHtml(system.architecture || '')}</td>
                        <td>${escapeHtml(system.ip || '')}</td>
                        <td>${system.updates_available ?
                            `<span class="update-badge update-available${system.pending_updates ? ' priority-' + getUpdatePriority(system.pending_updates) : ''}"
                                   title="Updates available${system.pending_updates ? ': ' + system.pending_updates.map(u => escapeHtml(u.name)).join(', ') : ' - Click for details'}">
                                Updates${system.pending_updates ? ` (${system.pending_updates.length})` : ' (click for details)'}
                                ${system.pending_updates && getUpdatePriority(system.pending_updates) === 'high' ? ' ⚠️' : ''}
                            </span>` :
                            `<span class="update-badge up-to-date">Up to date</span>`
                        }</td>
                        <td>${formatTimestamp(system.last_seen || '')}</td>
                    </tr>
                    <tr class="details-row" data-hostname="${escapeHtml(system.hostname)}" style="display: none;">
                        <td colspan="7">
                            <div class="details-content">Loading...</div>
                        </td>
                    </tr>`;
                    }
                )
                .filter(row => row !== '') // Remove empty rows from invalid systems
                .join("");
            
            systemsTable.innerHTML = rows;
        } catch (error) {
            console.error("Failed to generate table rows:", error);
            systemsTable.innerHTML = `
                <tr>
                    <td colspan="7" style="text-align: center; padding: 2rem; color: #f00;">
                        Error rendering systems: ${error.message}
                    </td>
                </tr>
            `;
            return;
        }

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
                console.error(`Failed to fetch system details for ${hostname}:`, error);
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

