// Theme management with localStorage and browser preference detection
(function() {
    const THEME_STORAGE_KEY = 'mqttfun-theme';
    const THEME_ATTR = 'data-theme';
    
    // Get the current theme from localStorage or detect from browser preference
    function getInitialTheme() {
        const storedTheme = localStorage.getItem(THEME_STORAGE_KEY);
        if (storedTheme === 'light' || storedTheme === 'dark') {
            return storedTheme;
        }
        
        // Detect browser preference
        if (window.matchMedia && window.matchMedia('(prefers-color-scheme: light)').matches) {
            return 'light';
        }
        
        // Default to dark mode
        return 'dark';
    }
    
    // Apply theme to document
    function applyTheme(theme) {
        if (theme === 'light') {
            document.documentElement.setAttribute(THEME_ATTR, 'light');
        } else {
            document.documentElement.removeAttribute(THEME_ATTR);
        }
        updateThemeToggleButton(theme);
    }
    
    // Update theme toggle button icon
    function updateThemeToggleButton(theme) {
        const toggleButton = document.getElementById('theme-toggle-btn');
        if (toggleButton) {
            toggleButton.textContent = theme === 'light' ? '🌙' : '☀️';
            toggleButton.setAttribute('aria-label', theme === 'light' ? 'Switch to dark mode' : 'Switch to light mode');
        }
    }
    
    // Toggle theme
    function toggleTheme() {
        const currentTheme = document.documentElement.getAttribute(THEME_ATTR) === 'light' ? 'light' : 'dark';
        const newTheme = currentTheme === 'light' ? 'dark' : 'light';
        
        localStorage.setItem(THEME_STORAGE_KEY, newTheme);
        applyTheme(newTheme);
    }
    
    // Initialize theme on page load
    function initTheme() {
        const theme = getInitialTheme();
        applyTheme(theme);
        
        // Listen for browser preference changes (only if user hasn't manually set a preference)
        if (window.matchMedia) {
            const mediaQuery = window.matchMedia('(prefers-color-scheme: light)');
            mediaQuery.addEventListener('change', (e) => {
                // Only update if user hasn't manually set a preference
                const storedTheme = localStorage.getItem(THEME_STORAGE_KEY);
                if (!storedTheme) {
                    const newTheme = e.matches ? 'light' : 'dark';
                    applyTheme(newTheme);
                }
            });
        }
        
        // Set up toggle button event listener
        const toggleButton = document.getElementById('theme-toggle-btn');
        if (toggleButton) {
            toggleButton.addEventListener('click', toggleTheme);
        }
    }
    
    // Run on DOMContentLoaded or immediately if already loaded
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initTheme);
    } else {
        initTheme();
    }
})();

