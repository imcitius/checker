// Dashboard UI Controller
document.addEventListener('DOMContentLoaded', () => {
    // Global state
    const state = {
        checks: [],
        filteredChecks: [],
        filters: {
            search: '',
            status: 'all',
            project: 'all',
            type: 'all'
        },
        websocket: null,
        projects: new Set(),
        types: new Set(),
        expandedRows: new Set() // Track which rows are expanded
    };

    // DOM Elements
    const elements = {
        connectionStatus: document.getElementById('connection-status'),
        searchInput: document.getElementById('search-input'),
        filterStatus: document.getElementById('filter-status'),
        filterProject: document.getElementById('filter-project'),
        filterType: document.getElementById('filter-type'),
        checksList: document.getElementById('checks-list'),
        stats: {
            total: document.querySelector('#total-checks .stat-value'),
            healthy: document.querySelector('#healthy-checks .stat-value'),
            unhealthy: document.querySelector('#unhealthy-checks .stat-value'),
            disabled: document.querySelector('#disabled-checks .stat-value'),
            silenced: document.querySelector('#silenced-checks .stat-value')
        },
        notificationArea: document.getElementById('notification-area')
    };

    // Templates
    const templates = {
        expandableRow: document.getElementById('expandable-row-template'),
        card: document.getElementById('card-template')
    };

    // Debugging for templates
    console.log('Templates loaded:', {
        'expandableRow': templates.expandableRow ? 'Found' : 'Missing',
        'card': templates.card ? 'Found' : 'Missing'
    });

    // Initialize WebSocket connection
    function initWebSocket() {
        // Get the host from the current URL
        const host = window.location.host;
        const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
        const wsUrl = `${protocol}//${host}/ws`;

        console.log('[WebSocket] Current page URL:', window.location.href);

        console.log(`[WebSocket] Initializing connection to ${wsUrl}`);

        // Close existing connection if any
        if (state.websocket) {
            console.log('[WebSocket] Closing existing connection');
            state.websocket.close();
        }

        try {
            // Create WebSocket connection
            console.log('[WebSocket] Creating new WebSocket instance to URL:', wsUrl);
            try {
                // Explicitly make a simple connection without any protocols
                state.websocket = new WebSocket(wsUrl);

                // Set binary type
                state.websocket.binaryType = 'arraybuffer';

                console.log('[WebSocket] Initial connection created, status:', state.websocket.readyState);
            } catch (wsError) {
                console.error('[WebSocket] Error creating WebSocket instance:', wsError);
                elements.connectionStatus.textContent = 'Connection error: ' + wsError.message;
                setTimeout(initWebSocket, 3000);
                return;
            }

            // Connection opened
            state.websocket.addEventListener('open', (event) => {
                console.log('[WebSocket] Connection established successfully');
                elements.connectionStatus.textContent = 'Connected to the server';
                elements.connectionStatus.classList.remove('disconnected');
                elements.connectionStatus.classList.add('connected');

                // Send an initial message to request data
                console.log('[WebSocket] Connected - sending initial request for data');
                try {
                    state.websocket.send(JSON.stringify({ action: 'getChecks' }));
                    console.log('[WebSocket] Initial request sent successfully');
                } catch (err) {
                    console.error('[WebSocket] Error sending initial request:', err);
                }
            });

            // Listen for messages
            state.websocket.addEventListener('message', (event) => {
                try {
                    // Parse message
                    let data = typeof event.data === 'object' ? event.data : JSON.parse(event.data);

                    if (data.type === 'checks') {
                        console.log('[WebSocket] Received checks update:', data.checks.length);

                        // Debug: Log the raw data structure of the first few checks
                        if (data.checks.length > 0) {
                            console.log('[DEBUG] Raw check data structure sample:');
                            data.checks.slice(0, 2).forEach((check, i) => {
                                console.log(`Check ${i}:`, JSON.stringify(check, null, 2));
                            });
                        }

                        // Normalize all checks
                        const validChecks = data.checks
                            .map(check => normalizeCheckData(check))
                            .filter(check => check !== null);

                        if (validChecks.length < data.checks.length) {
                            console.warn(`[WebSocket] Filtered out ${data.checks.length - validChecks.length} invalid checks`);
                        }

                        state.checks = validChecks;

                        // Update UI
                        try {
                            updateFilters();
                            applyFilters();
                            render();
                            elements.connectionStatus.textContent = `Connected: ${state.checks.length} checks loaded`;
                        } catch (uiError) {
                            console.error('[WebSocket] Error updating UI:', uiError);
                        }
                    } else if (data.type === 'update') {
                        // Handle single check update
                        if (!data.check) {
                            console.error('[WebSocket] Update message missing check data');
                            return;
                        }

                        const normalizedCheck = normalizeCheckData(data.check);
                        if (!normalizedCheck) {
                            console.error('[WebSocket] Failed to normalize update check data');
                            return;
                        }

                        // Update the check in state
                        const index = state.checks.findIndex(c => c.UUID === normalizedCheck.UUID);
                        if (index !== -1) {
                            state.checks[index] = normalizedCheck;
                        } else {
                            state.checks.push(normalizedCheck);
                        }

                        // Update UI
                        try {
                            updateFilters();
                            applyFilters();
                            render();
                        } catch (uiError) {
                            console.error('[WebSocket] Error updating UI after check update:', uiError);
                        }
                    } else if (data.type === 'ack') {
                        // Server acknowledgment message - just log it
                        console.log('[WebSocket] Received acknowledgment from server:', data);
                        // No additional processing needed
                    } else {
                        console.warn(`[WebSocket] Unknown message type: ${data.type}`);
                    }
                } catch (error) {
                    console.error('[WebSocket] Error processing message:', error);
                    console.log('[WebSocket] Raw message that caused error:', event.data);
                }
            });

            // Connection closed
            state.websocket.addEventListener('close', (event) => {
                console.log(`[WebSocket] Connection closed. Code: ${event.code}, Reason: ${event.reason || 'No reason provided'}`);
                console.log('[WebSocket] Last checks data count:', state.checks.length);

                // Update UI to show disconnection
                elements.connectionStatus.textContent = 'Disconnected from server. Attempting to reconnect...';
                elements.connectionStatus.classList.remove('connected');
                elements.connectionStatus.classList.add('disconnected');

                // If we have checks already loaded, keep them visible
                if (state.checks.length > 0) {
                    console.log('[WebSocket] Keeping existing checks visible despite disconnection');
                } else {
                    console.log('[WebSocket] No checks to display after disconnection');
                }

                // Try to reconnect after 2 seconds
                console.log('[WebSocket] Will attempt to reconnect in 2 seconds');
                setTimeout(initWebSocket, 2000);
            });

            // Connection error
            state.websocket.addEventListener('error', (event) => {
                console.error('[WebSocket] Connection error:', event);
                elements.connectionStatus.textContent = 'WebSocket connection error';
                elements.connectionStatus.classList.remove('connected');
                elements.connectionStatus.classList.add('disconnected');
            });
        } catch (error) {
            console.error('[WebSocket] Error creating connection:', error);
            elements.connectionStatus.textContent = 'Failed to create WebSocket connection: ' + error.message;
            elements.connectionStatus.classList.remove('connected');
            elements.connectionStatus.classList.add('disconnected');

            // For debugging - try accessing direct URL
            fetch(wsUrl.replace('ws:', 'http:').replace('wss:', 'https:'))
                .then(response => {
                    console.log('[Debug] HTTP request to WS endpoint response:', response.status);
                })
                .catch(err => {
                    console.error('[Debug] HTTP request to WS endpoint failed:', err);
                });

            // Try to reconnect after 5 seconds
            setTimeout(initWebSocket, 5000);
        }
    }

    // Update a single check in the state
    function updateCheck(updatedCheck) {
        if (!updatedCheck) {
            console.error('[UpdateCheck] Received invalid check:', updatedCheck);
            return;
        }

        console.log('[UpdateCheck] Updating check:', updatedCheck);

        // Normalize the properties to ensure consistent access regardless of case
        // This fixes issues where server sends camelCase but client expects PascalCase or vice versa
        updatedCheck.UUID = updatedCheck.UUID || updatedCheck.uuid;
        updatedCheck.Name = updatedCheck.Name || updatedCheck.name;
        updatedCheck.Project = updatedCheck.Project || updatedCheck.project;
        updatedCheck.CheckType = updatedCheck.CheckType || updatedCheck.check_type || updatedCheck.type || 'Unknown';
        updatedCheck.LastResult = updatedCheck.LastResult !== undefined ? updatedCheck.LastResult :
            (updatedCheck.lastResult !== undefined ? updatedCheck.lastResult : false);
        updatedCheck.LastExec = updatedCheck.LastExec || updatedCheck.lastExec || 'Unknown';
        updatedCheck.Enabled = updatedCheck.Enabled !== undefined ? updatedCheck.Enabled :
            (updatedCheck.enabled !== undefined ? updatedCheck.enabled : true);
        updatedCheck.Message = updatedCheck.Message || updatedCheck.message || '';
        updatedCheck.Host = updatedCheck.Host || updatedCheck.host || '';
        updatedCheck.Periodicity = updatedCheck.Periodicity || updatedCheck.periodicity || '';

        const uuid = updatedCheck.UUID;

        if (!uuid) {
            console.error('[UpdateCheck] Check missing UUID, cannot update');
            return;
        }

        // Check if there's a user-intended state for this check's enabled status
        // This prevents WebSocket updates from overriding user actions
        if (window.userIntendedStates && window.userIntendedStates[uuid] !== undefined) {
            const userIntendedState = window.userIntendedStates[uuid];
            const lastToggleTimestamp = window.lastToggleTimestamps ? window.lastToggleTimestamps[uuid] : 0;

            // Only respect user's intended state if the toggle was recent (within last 10 seconds)
            const isRecentToggle = lastToggleTimestamp && (Date.now() - lastToggleTimestamp < 10000);

            if (isRecentToggle && updatedCheck.Enabled !== userIntendedState) {
                console.log(`[UpdateCheck] Received server update with Enabled=${updatedCheck.Enabled} but respecting user's intended state=${userIntendedState}`);
                updatedCheck.Enabled = userIntendedState;
            } else if (!isRecentToggle) {
                // If the toggle is old, we can clear the user's intended state
                console.log(`[UpdateCheck] Clearing stale user intended state for ${uuid}`);
                delete window.userIntendedStates[uuid];
            }
        }

        // Log detailed information about the update
        console.log(`[UpdateCheck] Check details - UUID: ${uuid}, Name: ${updatedCheck.Name}, Enabled: ${updatedCheck.Enabled}`);

        const index = state.checks.findIndex(check => check.UUID === uuid);

        if (index !== -1) {
            console.log(`[UpdateCheck] Found existing check at index ${index}, updating from ${state.checks[index].Enabled} to ${updatedCheck.Enabled}`);
            state.checks[index] = updatedCheck;
        } else {
            console.log('[UpdateCheck] Adding new check');
            state.checks.push(updatedCheck);
        }

        // Update filtered checks list if the check is there
        const filteredIndex = state.filteredChecks.findIndex(check => check.UUID === uuid);
        if (filteredIndex !== -1) {
            console.log(`[UpdateCheck] Updating filtered check at index ${filteredIndex}`);
            state.filteredChecks[filteredIndex] = updatedCheck;
        }

        // After updating state, reapply filters and re-render
        updateFilters();
        applyFilters();
        render();
    }

    // Update the filter options based on available data
    function updateFilters() {
        // Reset project and type sets
        state.projects.clear();
        state.types.clear();

        console.log('[Filters] Updating filter options from', state.checks.length, 'checks');

        // Collect unique projects and types
        state.checks.forEach(check => {
            if (!check) return;

            // Handle different property naming conventions
            const project = check.Project || check.project || '';
            const checkType = check.CheckType || '';

            if (project) state.projects.add(project);
            if (checkType) state.types.add(checkType);
        });

        console.log('[Filters] Found unique projects:', Array.from(state.projects));
        console.log('[Filters] Found unique types:', Array.from(state.types));

        // Update project filter options
        elements.filterProject.innerHTML = '<option value="all">All projects</option>';
        state.projects.forEach(project => {
            const option = document.createElement('option');
            option.value = project;
            option.textContent = project;
            elements.filterProject.appendChild(option);
        });

        // Update type filter options
        elements.filterType.innerHTML = '<option value="all">All types</option>';
        state.types.forEach(type => {
            const option = document.createElement('option');
            option.value = type;
            option.textContent = type;
            elements.filterType.appendChild(option);
        });

        // Update stats
        updateStats();
    }

    // Update the stats display
    function updateStats() {
        console.log('[Stats] Updating dashboard statistics');
        const stats = {
            total: state.checks.length,
            healthy: state.checks.filter(check => {
                if (!check) return false;
                const lastResult = check.LastResult !== undefined ? check.LastResult : (check.lastResult !== undefined ? check.lastResult : false);
                const enabled = check.Enabled !== undefined ? check.Enabled : (check.enabled !== undefined ? check.enabled : true);
                return lastResult && enabled;
            }).length,
            unhealthy: state.checks.filter(check => {
                if (!check) return false;
                const lastResult = check.LastResult !== undefined ? check.LastResult : (check.lastResult !== undefined ? check.lastResult : false);
                const enabled = check.Enabled !== undefined ? check.Enabled : (check.enabled !== undefined ? check.enabled : true);
                return !lastResult && enabled;
            }).length,
            disabled: state.checks.filter(check => {
                if (!check) return false;
                const enabled = check.Enabled !== undefined ? check.Enabled : (check.enabled !== undefined ? check.enabled : true);
                return !enabled;
            }).length,
            silenced: state.checks.filter(check => {
                if (!check) return false;
                return check.IsSilenced === true;
            }).length
        };

        console.log('[Stats] Stats calculated:', stats);

        // Check if DOM elements exist before updating
        if (!elements.stats.total || !elements.stats.healthy || !elements.stats.unhealthy || !elements.stats.disabled || !elements.stats.silenced) {
            console.error('[Stats] One or more stat elements not found in DOM');
            return;
        }

        try {
            elements.stats.total.textContent = stats.total;
            elements.stats.healthy.textContent = stats.healthy;
            elements.stats.unhealthy.textContent = stats.unhealthy;
            elements.stats.disabled.textContent = stats.disabled;
            if (elements.stats.silenced) {
                elements.stats.silenced.textContent = stats.silenced;
            }
            console.log('[Stats] Updated DOM with stats');
        } catch (err) {
            console.error('[Stats] Error updating stats in DOM:', err);
        }
    }

    // Filter checks based on search and filter settings
    function applyFilters() {
        console.log('[Filters] Applying filters:', state.filters);
        console.log('[Filters] Total checks before filtering:', state.checks.length);

        state.filteredChecks = state.checks.filter(check => {
            if (!check) return false;

            // Handle different property naming conventions
            const name = (check.Name || check.name || '').toLowerCase();
            const project = (check.Project || check.project || '').toLowerCase();
            const healthcheck = (check.Healthcheck || check.healthcheck || '').toLowerCase();
            const checkType = check.CheckType || check.type || '';
            const lastResult = check.LastResult !== undefined ? check.LastResult : (check.lastResult !== undefined ? check.lastResult : false);
            const enabled = check.Enabled !== undefined ? check.Enabled : (check.enabled !== undefined ? check.enabled : true);

            // Search filter
            const searchTerm = state.filters.search.toLowerCase();
            const searchMatch = searchTerm === '' ||
                name.includes(searchTerm) ||
                project.includes(searchTerm) ||
                healthcheck.includes(searchTerm);

            // Status filter
            let statusMatch = true;
            if (state.filters.status === 'healthy') {
                statusMatch = lastResult && enabled;
            } else if (state.filters.status === 'unhealthy') {
                statusMatch = !lastResult && enabled;
            }

            // Project filter
            const projectMatch = state.filters.project === 'all' || project === state.filters.project.toLowerCase();

            // Type filter
            const typeMatch = state.filters.type === 'all' || checkType === state.filters.type;

            return searchMatch && statusMatch && projectMatch && typeMatch;
        });

        console.log('[Filters] Checks after filtering:', state.filteredChecks.length);
    }

    // Render the checks list with expandable rows
    function render() {
        console.log('[Render] Starting render with', state.filteredChecks.length, 'filtered checks');
        // Update stats before rendering
        updateStats();
        renderExpandableRows();
    }

    // Render the expandable rows view
    function renderExpandableRows() {
        console.log('[Render] Rendering expandable rows with', state.filteredChecks.length, 'checks');
        console.log('[Render] First filtered check:', state.filteredChecks.length > 0 ? JSON.stringify(state.filteredChecks[0], null, 2) : 'None');
        console.log('[Render] Templates available:', {
            expandableRow: templates.expandableRow ? 'Yes' : 'No',
            card: templates.card ? 'Yes' : 'No'
        });

        // Clear previous content
        if (elements.checksList) {
            console.log('[Render] Checks list element found, clearing content');
            elements.checksList.innerHTML = '';
        } else {
            console.error('[Render] Checks list element not found in the DOM');
            // Try to find it again
            elements.checksList = document.getElementById('checks-list');
            if (elements.checksList) {
                console.log('[Render] Found checks list element on retry');
                elements.checksList.innerHTML = '';
            } else {
                // Try alternative selectors
                const alternativeList = document.querySelector('tbody#checks-list') ||
                    document.querySelector('#checks-list') ||
                    document.querySelector('.checks-container tbody');
                if (alternativeList) {
                    console.log('[Render] Found checks list with alternative selector');
                    elements.checksList = alternativeList;
                    elements.checksList.innerHTML = '';
                } else {
                    // Add a visible error message
                    const container = document.querySelector('.dashboard-container');
                    if (container) {
                        const errorEl = document.createElement('div');
                        errorEl.className = 'error-message';
                        errorEl.style.cssText = 'color: red; padding: 20px; text-align: center; background: #fff3cd; margin: 20px; border-radius: 4px;';
                        errorEl.textContent = 'Error: Cannot find the checks list element in the DOM.';
                        container.appendChild(errorEl);
                    }
                    return;
                }
            }
        }

        if (!state.filteredChecks || state.filteredChecks.length === 0) {
            console.log('No checks to display');
            const emptyRow = document.createElement('tr');
            emptyRow.innerHTML = '<td colspan="9" style="text-align: center; padding: 20px;">No checks available</td>';
            elements.checksList.appendChild(emptyRow);
            return;
        }

        // Log data structure for debugging
        console.log('[Debug] Check data structure sample:', JSON.stringify(state.filteredChecks[0], null, 2));

        state.filteredChecks.forEach((check, index) => {
            try {
                // Verify check has required properties
                if (!check || typeof check !== 'object') {
                    console.error('Invalid check object at index', index, ':', check);
                    return;
                }

                // Check for UUID
                const uuid = check.UUID || check.uuid;
                if (!uuid) {
                    console.error('Check missing UUID at index', index, ':', check);
                    return;
                }

                console.log(`Rendering check ${index}:`, check.Name || check.name || 'Unnamed check', 'UUID:', uuid);

                // Clone template
                if (!templates.expandableRow) {
                    console.error('Expandable row template not found');

                    // Debug template status
                    console.error('Template status:', {
                        'expandableRow': document.getElementById('expandable-row-template') ? 'Found in DOM but not in templates' : 'Not found in DOM',
                        'templates object': JSON.stringify(templates)
                    });
                    return;
                }

                let mainRow, detailsRow;
                try {
                    const template = templates.expandableRow.content.cloneNode(true);
                    const rows = template.querySelectorAll('tr');
                    mainRow = rows[0]; // First row - main content
                    detailsRow = rows[1]; // Second row - details with UUID and URL

                    console.log('Successfully cloned template rows:', {
                        mainRow: mainRow ? 'Found' : 'Missing',
                        detailsRow: detailsRow ? 'Found' : 'Missing'
                    });
                } catch (err) {
                    console.error('Error cloning template:', err);
                    console.log('Template content:', templates.expandableRow ? templates.expandableRow.innerHTML : 'Missing template');
                    return;
                }

                // Set row ID for tracking expanded state
                const rowId = `check-${uuid}`;
                mainRow.id = rowId;

                // Determine status class
                const enabled = check.Enabled !== undefined ? check.Enabled : (check.enabled !== undefined ? check.enabled : true);
                const lastResult = check.LastResult !== undefined ? check.LastResult : (check.lastResult !== undefined ? check.lastResult : false);

                // Set the row class based on status - disabled checks should have 'disabled' class
                mainRow.className = `check-row ${enabled ? (lastResult ? 'healthy' : 'unhealthy') : 'disabled'}`;

                // Apply the same status class to the details row for consistent styling
                if (detailsRow) {
                    detailsRow.className = `check-details-row always-visible ${enabled ? (lastResult ? 'healthy' : 'unhealthy') : 'disabled'}`;
                }

                // Set check data with fallbacks for different property naming conventions
                const nameEl = mainRow.querySelector('.check-name');
                if (nameEl) {
                    nameEl.textContent = check.Name || check.name || 'Unnamed';
                } else {
                    console.error('Could not find .check-name element');
                }

                const projectEl = mainRow.querySelector('.check-project');
                if (projectEl) {
                    projectEl.textContent = check.Project || check.project || 'N/A';
                } else {
                    console.error('Could not find .check-project element');
                }

                const typeEl = mainRow.querySelector('.check-type');
                if (typeEl) {
                    typeEl.textContent = check.CheckType || check.type || 'Unknown';
                } else {
                    console.error('Could not find .check-type element');
                }

                const statusIndicator = mainRow.querySelector('.check-status-indicator');
                if (statusIndicator) {
                    // For disabled checks, use a 'disabled' class for the indicator rather than unhealthy
                    if (!enabled) {
                        statusIndicator.className = 'check-status-indicator disabled';
                    } else {
                        statusIndicator.className = `check-status-indicator ${lastResult ? 'healthy' : 'unhealthy'}`;
                    }
                } else {
                    console.error('Could not find .check-status-indicator element');
                }

                const statusTextEl = mainRow.querySelector('.check-status-text');
                const isSilenced = check.IsSilenced || false;
                if (statusTextEl) {
                    // Set status text appropriately for disabled checks
                    if (!enabled) {
                        statusTextEl.textContent = 'Disabled';
                    } else if (isSilenced) {
                        statusTextEl.innerHTML = (lastResult ? 'Healthy' : 'Unhealthy') + ' <span class="silenced-badge" title="Alerts are silenced for this check">SILENCED</span>';
                    } else {
                        statusTextEl.textContent = lastResult ? 'Healthy' : 'Unhealthy';
                    }
                } else {
                    console.error('Could not find .check-status-text element');
                }

                const lastExecEl = mainRow.querySelector('.check-last-exec');
                if (lastExecEl) {
                    lastExecEl.textContent = check.LastExec || check.lastExec || 'Never';
                } else {
                    console.error('Could not find .check-last-exec element');
                }

                // Set UUID value in details row
                const uuidValueEl = detailsRow.querySelector('.uuid-value');
                if (uuidValueEl) {
                    uuidValueEl.textContent = uuid || 'Unknown';

                    // Make UUID cell clickable for copying
                    const uuidContainer = detailsRow.querySelector('.uuid-container');
                    if (uuidContainer) {
                        uuidContainer.addEventListener('click', function (e) {
                            e.stopPropagation();

                            // Copy UUID to clipboard
                            navigator.clipboard.writeText(uuid).then(() => {
                                // Show success feedback
                                this.classList.add('copied');
                                const originalTitle = this.getAttribute('title');
                                this.setAttribute('title', 'Copied!');

                                // Reset after 2 seconds
                                setTimeout(() => {
                                    this.classList.remove('copied');
                                    this.setAttribute('title', originalTitle);
                                }, 2000);
                            }).catch(err => {
                                console.error('Failed to copy UUID:', err);
                            });
                        });
                    }
                }

                // Set URL value with link if it's an HTTP check in details row
                const urlValueEl = detailsRow.querySelector('.url-value');
                if (urlValueEl) {
                    const checkType = (check.CheckType || check.type || '').toLowerCase();
                    const url = check.URL || check.url || check.Url || '';

                    if (checkType.includes('http') && url) {
                        // Format URL and make clickable
                        let formattedUrl = url;
                        if (!url.startsWith('http')) {
                            formattedUrl = 'http://' + url;
                        }
                        urlValueEl.innerHTML = `<a href="${formattedUrl}" target="_blank" title="${formattedUrl}">${url}</a>`;
                    } else if (checkType.includes('http')) {
                        urlValueEl.textContent = 'Not specified';
                        urlValueEl.classList.add('text-muted');
                    } else {
                        // For non-HTTP checks, use Host if available
                        const host = check.Host || check.host || '';
                        urlValueEl.textContent = host || 'N/A';
                    }
                }

                const frequencyValue = mainRow.querySelector('.frequency-value');
                if (frequencyValue) {
                    const duration = check.Periodicity || check.periodicity || check.Duration || check.duration || 'Not set';
                    console.log('[Render] Setting frequency value:', duration);
                    frequencyValue.textContent = duration;
                    frequencyValue.title = duration;
                } else {
                    console.warn('[Render] Frequency value element not found');
                }

                // Add event listener for toggle switch
                const checkToggle = mainRow.querySelector('.check-toggle');
                if (checkToggle) {
                    // Set initial checked state
                    checkToggle.checked = enabled;
                    // Store the UUID directly on the element for easier access
                    checkToggle.dataset.uuid = uuid;

                    // Remove existing event listeners to prevent duplicates
                    const oldElement = checkToggle.cloneNode(true);
                    checkToggle.parentNode.replaceChild(oldElement, checkToggle);
                    const newCheckToggle = oldElement;

                    // Re-establish data-uuid attribute
                    newCheckToggle.dataset.uuid = uuid;
                    newCheckToggle.checked = enabled;

                    // Add debounced event listener to prevent bounce-back
                    newCheckToggle.addEventListener('change', (e) => {
                        e.stopPropagation(); // Prevent row click

                        // Disable the toggle temporarily to prevent multiple clicks
                        newCheckToggle.disabled = true;

                        // Add visual feedback when toggling
                        const actionsContainer = mainRow.querySelector('.actions-container');
                        if (actionsContainer) {
                            // Add a temporary processing indicator
                            const processingIndicator = document.createElement('span');
                            processingIndicator.className = 'processing-indicator';
                            processingIndicator.textContent = '⟳';
                            processingIndicator.style.cssText = 'animation: spin 1s linear infinite; display: inline-block; margin-left: 5px; color: #0066cc;';
                            actionsContainer.appendChild(processingIndicator);

                            // Add keyframes for spin animation if not already added
                            if (!document.querySelector('style#spin-animation')) {
                                const styleEl = document.createElement('style');
                                styleEl.id = 'spin-animation';
                                styleEl.textContent = '@keyframes spin { 0% { transform: rotate(0deg); } 100% { transform: rotate(360deg); } }';
                                document.head.appendChild(styleEl);
                            }
                        }

                        // Get the current checked state
                        const isChecked = e.target.checked;
                        console.log(`[Toggle Event] Toggle changed to ${isChecked} for ${uuid}`);

                        // Call toggle check with small delay to let UI update first
                        setTimeout(() => {
                            toggleCheck(uuid, isChecked);
                            // Re-enable after a delay
                            setTimeout(() => {
                                newCheckToggle.disabled = false;
                                // Remove processing indicator
                                const processingIndicator = mainRow.querySelector('.processing-indicator');
                                if (processingIndicator) {
                                    processingIndicator.remove();
                                }
                            }, 500);
                        }, 10);
                    });
                }

                // Add event listener to the edit button
                const editButton = mainRow.querySelector('.edit-button');
                if (editButton) {
                    // Remove existing event listeners to prevent duplicates
                    const oldEditButton = editButton.cloneNode(true);
                    editButton.parentNode.replaceChild(oldEditButton, editButton);
                    const newEditButton = oldEditButton;

                    // Add click event listener
                    newEditButton.addEventListener('click', (e) => {
                        e.stopPropagation(); // Prevent row click
                        e.preventDefault();

                        // Navigate to check definitions page with this UUID
                        window.location.href = `/check-definitions?uuid=${uuid}`;
                    });
                }

                // Append both rows to the DOM
                elements.checksList.appendChild(mainRow);
                if (detailsRow) {
                    elements.checksList.appendChild(detailsRow);
                }

            } catch (error) {
                console.error(`Error rendering check ${index}:`, error, '\nCheck data:', check);
            }
        });
    }

    // Render the detailed view (cards)
    function renderDetailedView() {
        const container = document.getElementById('detailed-view');
        container.innerHTML = '';

        state.filteredChecks.forEach(check => {
            const clone = templates.card.content.cloneNode(true);
            const card = clone.querySelector('.check-card');

            // Set card status class
            if (!check.Enabled) {
                card.classList.add('disabled');
            } else if (check.LastResult) {
                card.classList.add('healthy');
            } else {
                card.classList.add('unhealthy');
            }

            // Fill in the card data
            clone.querySelector('.check-name').textContent = check.Name;
            clone.querySelector('.check-status').classList.add(check.LastResult ? 'healthy' : 'unhealthy');
            clone.querySelector('.check-toggle').checked = check.Enabled;
            clone.querySelector('.check-toggle').setAttribute('onchange', `toggleCheck('${check.UUID}', this.checked)`);
            clone.querySelector('.check-project').textContent = check.Project;
            clone.querySelector('.check-type').textContent = check.CheckType || 'Unknown';
            clone.querySelector('.check-group').textContent = check.Healthcheck;
            clone.querySelector('.check-last-exec').textContent = check.LastExec;
            clone.querySelector('.check-last-ping').textContent = check.LastPing;
            clone.querySelector('.check-uuid').textContent = `UUID: ${check.UUID}`;

            // Add host and periodicity if available
            const detailsSection = clone.querySelector('.check-details-section');
            if (check.Host || check.Periodicity) {
                // Create container for host and periodicity if it doesn't exist
                if (!detailsSection) {
                    const detailsContainer = document.createElement('div');
                    detailsContainer.className = 'check-details-section';
                    detailsContainer.style.marginTop = '10px';

                    // Create a single details element with both host and periodicity
                    const detailsEl = document.createElement('div');
                    detailsEl.className = 'check-details';

                    let detailsContent = '';
                    if (check.Host) {
                        detailsContent += `<span class="check-host"><i>🖥️</i>Host: ${check.Host}</span>`;
                    }

                    if (check.Periodicity) {
                        detailsContent += `<span class="check-periodicity"><i>🔄</i>Every ${check.Periodicity}</span>`;
                    }

                    detailsEl.innerHTML = detailsContent;
                    detailsContainer.appendChild(detailsEl);

                    // Insert after the check name
                    const nameEl = clone.querySelector('.check-name');
                    nameEl.parentNode.insertBefore(detailsContainer, nameEl.nextSibling);
                }
            }

            // Add error message if available
            const messageEl = clone.querySelector('.check-message');
            if (check.Message && !check.LastResult) {
                messageEl.textContent = check.Message;
            } else {
                messageEl.style.display = 'none';
            }

            container.appendChild(clone);
        });
    }

    // Render the table view (full details)
    function renderTableView() {
        const tbody = document.getElementById('table-checks');
        tbody.innerHTML = '';

        state.filteredChecks.forEach(check => {
            const tr = document.createElement('tr');
            tr.className = check.Enabled ? (check.LastResult ? 'healthy' : 'unhealthy') : 'disabled';

            // Create check name cell with detailed info
            const nameWithDetails = `
                ${check.Name}
                ${(check.Host || check.Periodicity) ? `
                <span class="check-details">
                    ${check.Host ? `<span class="check-host"><i>🖥️</i>Host: ${check.Host}</span>` : ''}
                    ${check.Periodicity ? `<span class="check-periodicity"><i>🔄</i>Every ${check.Periodicity}</span>` : ''}
                </span>
                ` : ''}
            `;

            tr.innerHTML = `
                <td title="${check.Name}">${nameWithDetails}</td>
                <td>${check.Project}</td>
                <td>${check.Healthcheck}</td>
                <td>${check.CheckType || 'Unknown'}</td>
                <td>
                    <span class="check-status ${check.LastResult ? 'healthy' : 'unhealthy'}"></span>
                    ${check.LastResult ? 'Healthy' : 'Unhealthy'}
                </td>
                <td>${check.Message || '-'}</td>
                <td>${check.LastExec}</td>
                <td>${check.LastPing}</td>
                <td>
                    <label class="switch">
                        <input type="checkbox" ${check.Enabled ? 'checked' : ''} 
                               onchange="toggleCheck('${check.UUID}', this.checked)">
                        <span class="slider"></span>
                    </label>
                </td>
            `;

            tbody.appendChild(tr);
        });
    }

    // Toggle check enabled status
    function toggleCheck(uuid, enabled) {
        console.log(`[Toggle] Attempting to set check ${uuid} to enabled=${enabled}`);

        // Track this toggle operation with a unique timestamp
        const toggleTimestamp = Date.now();
        console.log(`[Toggle ${toggleTimestamp}] Starting toggle operation`);

        // Determine current state before updating
        const check = state.checks.find(c => c.UUID === uuid);
        const previousState = check ? check.Enabled : !enabled;

        // Check if we're actually changing the state
        if (check && previousState === enabled) {
            console.log(`[Toggle ${toggleTimestamp}] Check ${check.Name} (${uuid}) already in requested state: ${enabled}, skipping update`);
            return; // Skip if already in desired state
        }

        // Create a last toggle timestamp to track the latest toggle for this UUID
        if (!window.lastToggleTimestamps) {
            window.lastToggleTimestamps = {};
        }
        window.lastToggleTimestamps[uuid] = toggleTimestamp;

        // Store the user's intended state for this toggle operation
        if (!window.userIntendedStates) {
            window.userIntendedStates = {};
        }
        window.userIntendedStates[uuid] = enabled;

        // Immediately update local state for responsive UI
        if (check) {
            console.log(`[Toggle ${toggleTimestamp}] Updating local state for ${check.Name} (${uuid}) from ${check.Enabled} to ${enabled}`);
            check.Enabled = enabled;

            // Update filtered checks if the check is in the filtered list
            const filteredCheck = state.filteredChecks.find(c => c.UUID === uuid);
            if (filteredCheck) {
                filteredCheck.Enabled = enabled;
            }

            // Force update any checkboxes in the UI to match our state
            const allCheckboxes = document.querySelectorAll(`input[data-uuid="${uuid}"]`);
            allCheckboxes.forEach(checkbox => {
                checkbox.checked = enabled;
            });

            // Re-apply filters and render to update UI immediately
            applyFilters();
            render();
        } else {
            console.warn(`[Toggle ${toggleTimestamp}] Could not find check with UUID ${uuid} in local state, creating placeholder`);

            // Create a placeholder check if it doesn't exist in our local state
            const placeholderCheck = {
                UUID: uuid,
                Name: `Check ${uuid.substring(0, 8)}...`,
                Project: "Unknown",
                CheckType: "Unknown",
                LastResult: false,
                LastExec: "Never",
                Enabled: enabled,
                Message: "",
                Host: "",
                Periodicity: ""
            };

            // Add to state
            state.checks.push(placeholderCheck);

            // Re-apply filters and render to update UI immediately
            applyFilters();
            render();
        }

        // Send toggle request to server
        console.log(`[Toggle ${toggleTimestamp}] Sending API request to server`);
        fetch('/api/toggle-check', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/x-www-form-urlencoded',
            },
            body: `uuid=${uuid}&enabled=${enabled}`
        })
            .then(response => {
                if (!response.ok) {
                    return response.text().then(text => {
                        throw new Error(`Network response was not ok: ${response.status} ${response.statusText} - ${text}`);
                    });
                }
                console.log(`[Toggle ${toggleTimestamp}] Server accepted toggle for ${uuid} to enabled=${enabled}`);
                // Server will send an update via WebSocket to confirm

                // Show a success notification
                const checkName = check ? check.Name : `Check ${uuid.substring(0, 8)}...`;
                showNotification(`${checkName} ${enabled ? 'enabled' : 'disabled'} successfully`);

                return response.json();
            })
            .then(data => {
                console.log(`[Toggle ${toggleTimestamp}] Server response:`, data);

                // Check if the server response includes the enabled state
                if (data && data.enabled !== undefined && data.enabled !== enabled) {
                    console.warn(`[Toggle ${toggleTimestamp}] Server returned different enabled state (${data.enabled}) than requested (${enabled})`);

                    // Update our local state to match the server
                    const checkNow = state.checks.find(c => c.UUID === uuid);
                    if (checkNow) {
                        checkNow.Enabled = data.enabled;

                        // Update filtered checks
                        const filteredCheck = state.filteredChecks.find(c => c.UUID === uuid);
                        if (filteredCheck) {
                            filteredCheck.Enabled = data.enabled;
                        }

                        // Force the UI to update
                        applyFilters();
                        render();

                        // Update any checkboxes in the UI
                        const allCheckboxes = document.querySelectorAll(`input[data-uuid="${uuid}"]`);
                        allCheckboxes.forEach(checkbox => {
                            checkbox.checked = data.enabled;
                        });
                    }

                    // Update user intended state
                    window.userIntendedStates[uuid] = data.enabled;
                }

                // The server should send a WebSocket update, but we'll verify our UI state after a delay
                // as a backup mechanism
                setTimeout(() => {
                    // Only proceed if this is still the latest toggle operation for this UUID
                    if (window.lastToggleTimestamps[uuid] === toggleTimestamp) {
                        // Verify that our UI matches what we expect
                        const checkNow = state.checks.find(c => c.UUID === uuid);
                        if (checkNow && checkNow.Enabled !== enabled) {
                            console.log(`[Toggle ${toggleTimestamp}] UI state verification failed. Forcing update to ${enabled}`);
                            // Force the state to match what we sent to the server
                            checkNow.Enabled = enabled;

                            // Update filtered checks
                            const filteredCheck = state.filteredChecks.find(c => c.UUID === uuid);
                            if (filteredCheck) {
                                filteredCheck.Enabled = enabled;
                            }

                            // Force the UI to update
                            applyFilters();
                            render();

                            // Update any checkboxes in the UI
                            const allCheckboxes = document.querySelectorAll(`input[data-uuid="${uuid}"]`);
                            allCheckboxes.forEach(checkbox => {
                                console.log(`[Toggle ${toggleTimestamp}] Setting checkbox UI state to ${enabled}`);
                                checkbox.checked = enabled;
                            });
                        }
                    }
                }, 1500); // Wait 1.5 seconds to allow for WebSocket update to arrive
            })
            .catch(error => {
                console.error(`[Toggle ${toggleTimestamp}] Error:`, error);

                // Only process this error if this is still the latest toggle operation for this UUID
                if (window.lastToggleTimestamps[uuid] !== toggleTimestamp) {
                    console.log(`[Toggle ${toggleTimestamp}] Ignoring error because a newer toggle operation exists`);
                    return;
                }

                // Revert the local state change only if we're not currently in sync with server
                // This prevents excessive toggling
                if (check && check.Enabled !== previousState) {
                    console.log(`[Toggle ${toggleTimestamp}] Reverting local state for ${check.Name} (${uuid}) back to ${previousState}`);
                    check.Enabled = previousState;

                    // Update filtered checks if the check is in the filtered list
                    const filteredCheck = state.filteredChecks.find(c => c.UUID === uuid);
                    if (filteredCheck) {
                        filteredCheck.Enabled = previousState;
                    }

                    // Re-apply filters and render to update UI with reverted state
                    applyFilters();
                    render();

                    // Update any checkboxes in the UI
                    const allCheckboxes = document.querySelectorAll(`input[data-uuid="${uuid}"]`);
                    allCheckboxes.forEach(checkbox => {
                        console.log(`[Toggle ${toggleTimestamp}] Reverting checkbox UI state to ${previousState}`);
                        checkbox.checked = previousState;
                    });
                }

                // Show an error message to the user
                const errorMessage = `Failed to toggle check: ${error.message}`;
                console.error(errorMessage);

                // Display error using notification function
                showNotification(errorMessage, 'error');
            });
    }

    // Set up event listeners
    function setupEventListeners() {
        // Search input
        elements.searchInput.addEventListener('input', () => {
            state.filters.search = elements.searchInput.value;
            applyFilters();
            render();
        });

        // Filter changes
        elements.filterStatus.addEventListener('change', () => {
            state.filters.status = elements.filterStatus.value;
            applyFilters();
            render();
        });

        elements.filterProject.addEventListener('change', () => {
            state.filters.project = elements.filterProject.value;
            applyFilters();
            render();
        });

        elements.filterType.addEventListener('change', () => {
            state.filters.type = elements.filterType.value;
            applyFilters();
            render();
        });
    }

    // Initialize the dashboard
    function init() {
        console.log('[Init] Initializing dashboard');
        console.log('[Init] Page URL:', window.location.href);
        console.log('[Init] Document ready state:', document.readyState);

        // Fix for templates not being initialized correctly
        if (!templates.expandableRow) {
            console.log('[Init] Retrying template acquisition');
            templates.expandableRow = document.getElementById('expandable-row-template');
            console.log('[Init] expandableRow template:', templates.expandableRow ? 'Found' : 'Missing');
        }

        if (!templates.card) {
            templates.card = document.getElementById('card-template');
            console.log('[Init] card template:', templates.card ? 'Found' : 'Missing');
        }

        // Verify DOM elements
        let missingElements = [];
        for (const key in elements) {
            if (typeof elements[key] === 'object' && elements[key] !== null) {
                if (key === 'stats') {
                    for (const statKey in elements.stats) {
                        if (elements.stats[statKey] === null) {
                            missingElements.push(`stats.${statKey}`);
                            // Try to recover missing stats elements
                            if (document.querySelector(`#${statKey}-checks .stat-value`)) {
                                elements.stats[statKey] = document.querySelector(`#${statKey}-checks .stat-value`);
                                console.log(`[Recovery] Found missing stat element: ${statKey}`);
                            }
                        }
                    }
                }
            } else if (elements[key] === null) {
                missingElements.push(key);
                // Try to recover missing elements by ID
                elements[key] = document.getElementById(key);
                if (elements[key]) {
                    console.log(`[Recovery] Found missing element: ${key}`);
                } else {
                    // Try alternative selectors
                    if (key === 'checksList') {
                        elements[key] = document.querySelector('tbody#checks-list') || document.querySelector('#checks-list');
                        if (elements[key]) console.log('[Recovery] Found checks list with alternative selector');
                    }
                }
            }
        }

        if (missingElements.length > 0) {
            console.error(`[Init] Missing DOM elements: ${missingElements.join(', ')}`);
            // Add visible error to the page
            if (document.body) {
                const errorDiv = document.createElement('div');
                errorDiv.style.cssText = 'position:fixed; top:0; left:0; right:0; background-color:rgba(255,0,0,0.8); color:white; padding:10px; z-index:9999; text-align:center;';
                errorDiv.innerHTML = `<strong>Missing elements: ${missingElements.join(', ')}</strong>`;
                document.body.appendChild(errorDiv);
            }
        }

        // Initialize WebSocket and event listeners
        initWebSocket();
        setupEventListeners();

        // Set initial UI state
        if (elements.connectionStatus) {
            elements.connectionStatus.textContent = 'Connecting to server...';
            elements.connectionStatus.className = 'connection-status';
        } else {
            console.error('[Init] Cannot update connection status - element missing');
        }

        // Run health check after slight delay
        setTimeout(healthCheck, 3000);

        console.log('[Init] Dashboard initialization complete');
    }

    // Start the app
    init();
});

// Expose toggleCheck for global access
window.toggleCheck = function (uuid, enabled) {
    // This function is defined inside the DOMContentLoaded event handler,
    // but also exposed globally for use in inline event handlers
    fetch('/api/toggle-check', {
        method: 'POST',
        headers: {
            'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: `uuid=${uuid}&enabled=${enabled}`
    })
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok');
            }
            // The update will come via WebSocket
        })
        .catch(error => {
            console.error('Error:', error);
            // Revert the toggle if there was an error
            const checkboxes = document.querySelectorAll(`input[onchange*="'${uuid}'"]:checked`);
            checkboxes.forEach(checkbox => {
                checkbox.checked = !enabled;
            });
        });
};

// Function to create a check row from template
function createCheckRow(check) {
    console.log('[createCheckRow] Creating row for check:', {
        UUID: check.UUID,
        Name: check.Name,
        Host: check.Host,
        Periodicity: check.Periodicity
    });

    const template = document.getElementById('expandable-row-template');
    const row = template.content.cloneNode(true);

    // Set check name and project
    const nameElement = row.querySelector('.check-name');
    const projectElement = row.querySelector('.check-project');
    if (nameElement && projectElement) {
        nameElement.textContent = check.Name || 'Unnamed Check';
        projectElement.textContent = check.Project || 'No Project';
    }

    // Set frequency/duration value with proper fallbacks
    const frequencyValueElement = row.querySelector('.frequency-value');
    if (frequencyValueElement) {
        const duration = check.Periodicity || check.periodicity || check.Duration || check.duration || 'Not set';
        console.log('[createCheckRow] Setting frequency value:', duration);
        frequencyValueElement.textContent = formatDuration(duration);
        frequencyValueElement.title = formatDuration(duration);
    } else {
        console.warn('[createCheckRow] Frequency value element not found');
    }

    // Set status and result
    const statusElement = row.querySelector('.check-status');
    if (statusElement) {
        const isHealthy = check.LastResult !== undefined ? check.LastResult : true;
        statusElement.classList.toggle('healthy', isHealthy);
        statusElement.classList.toggle('unhealthy', !isHealthy);
        statusElement.textContent = isHealthy ? 'Healthy' : 'Unhealthy';
    }

    // Set last execution time
    const lastExecElement = row.querySelector('.last-execution');
    if (lastExecElement && check.LastExec) {
        lastExecElement.textContent = formatDate(check.LastExec);
        lastExecElement.title = new Date(check.LastExec).toLocaleString();
    }

    // Set message in details
    const messageElement = row.querySelector('.check-message');
    if (messageElement) {
        messageElement.textContent = check.Message || 'No message available';
        if (!check.LastResult) {
            messageElement.classList.add('error-message');
        }
    }

    return row;
}

// Helper function to format duration
function formatDuration(duration) {
    if (!duration) return 'Not set';

    // If duration is already formatted (e.g., "1h", "30m"), return as is
    if (typeof duration === 'string' && /^(\d+[hms])+$/.test(duration)) {
        return duration;
    }

    try {
        // Try to parse as a number (assuming seconds)
        const seconds = parseInt(duration);
        if (isNaN(seconds)) return duration;

        const hours = Math.floor(seconds / 3600);
        const minutes = Math.floor((seconds % 3600) / 60);
        const remainingSeconds = seconds % 60;

        const parts = [];
        if (hours > 0) parts.push(`${hours}h`);
        if (minutes > 0) parts.push(`${minutes}m`);
        if (remainingSeconds > 0) parts.push(`${remainingSeconds}s`);

        return parts.join('') || '0s';
    } catch (e) {
        console.warn('[formatDuration] Error formatting duration:', e);
        return duration;
    }
}

// Helper function to format dates
function formatDate(dateStr) {
    if (!dateStr) return 'Never';
    try {
        const date = new Date(dateStr);
        if (isNaN(date.getTime())) return dateStr;

        const now = new Date();
        const diff = now - date;

        // If less than 24 hours ago, show relative time
        if (diff < 24 * 60 * 60 * 1000) {
            const hours = Math.floor(diff / (60 * 60 * 1000));
            const minutes = Math.floor((diff % (60 * 60 * 1000)) / (60 * 1000));
            if (hours > 0) {
                return `${hours}h ${minutes}m ago`;
            }
            if (minutes > 0) {
                return `${minutes}m ago`;
            }
            return 'Just now';
        }

        // Otherwise show date and time
        return date.toLocaleString();
    } catch (e) {
        return dateStr;
    }
}

function normalizeCheckData(check) {
    if (!check || typeof check !== 'object') {
        console.warn('[normalizeCheckData] Invalid check object:', check);
        return null;
    }

    // Log original data
    console.log('[normalizeCheckData] Original check data:', {
        UUID: check.UUID || check.uuid,
        Name: check.Name || check.name,
        Host: check.Host || check.host,
        URL: check.URL || check.url || check.Url,
        Type: check.CheckType || check.check_type || check.type,
        Duration: check.Duration || check.duration || check.Periodicity || check.periodicity,
        raw: check
    });

    // Get the check type (lowercase for easier comparison)
    const checkType = (check.CheckType || check.check_type || check.type || '').toLowerCase();

    // For HTTP checks, prefer URL over Host
    let connectionValue = '';
    if (checkType.includes('http')) {
        connectionValue = check.URL || check.url || check.Url || check.Host || check.host || '';
    } else {
        connectionValue = check.Host || check.host || '';
    }

    // Create normalized check object
    const normalized = {
        UUID: check.UUID || check.uuid,
        Name: check.Name || check.name || 'Unnamed Check',
        Project: check.Project || check.project || 'No Project',
        CheckType: check.CheckType || check.check_type || check.type || 'Unknown',
        LastResult: check.LastResult !== undefined ? check.LastResult :
            (check.lastResult !== undefined ? check.lastResult : false),
        LastExec: check.LastExec || check.lastExec || 'Unknown',
        Enabled: check.Enabled !== undefined ? check.Enabled :
            (check.enabled !== undefined ? check.enabled : true),
        Message: check.Message || check.message || '',
        Host: connectionValue,
        URL: check.URL || check.url || check.Url || '',
        Periodicity: check.Periodicity || check.periodicity || check.Duration || check.duration || '',
        IsSilenced: check.IsSilenced !== undefined ? check.IsSilenced :
            (check.is_silenced !== undefined ? check.is_silenced : false)
    };

    // Log normalized data
    console.log('[normalizeCheckData] Normalized check data:', normalized);

    return normalized;
}

// Show a notification message in the notification area
function showNotification(message, type = 'info') {
    if (!elements.notificationArea) return;

    // Clear existing notifications
    elements.notificationArea.innerHTML = '';

    // Set message and class
    elements.notificationArea.textContent = message;
    elements.notificationArea.className = `notification-area ${type}`;

    // Auto-hide after 3 seconds for success messages
    if (type === 'success') {
        setTimeout(() => {
            elements.notificationArea.textContent = '';
            elements.notificationArea.className = 'notification-area';
        }, 3000);
    }
}