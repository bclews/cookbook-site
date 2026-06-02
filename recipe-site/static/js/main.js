// Main JavaScript functionality
(function() {
    'use strict';

    // Initialize when DOM is ready
    function init() {
        initIngredientCheckboxes();
        initPrintStyles();
        initLazyLoading();
        initDarkModeToggle();
    }

    // Allow checking off ingredients
    function initIngredientCheckboxes() {
        const checkboxes = document.querySelectorAll('.ingredient-checkbox');

        checkboxes.forEach(checkbox => {
            checkbox.addEventListener('change', function() {
                const item = this.closest('.ingredient-item');
                if (this.checked) {
                    item.classList.add('checked');
                } else {
                    item.classList.remove('checked');
                }
            });
        });
    }

    // Add print-specific handling
    function initPrintStyles() {
        window.addEventListener('beforeprint', function() {
            // Uncheck all ingredients before printing
            document.querySelectorAll('.ingredient-checkbox').forEach(cb => {
                cb.checked = false;
            });
            document.querySelectorAll('.ingredient-item').forEach(item => {
                item.classList.remove('checked');
            });
        });
    }

    // Lazy load images that are below the fold
    function initLazyLoading() {
        if ('IntersectionObserver' in window) {
            const imageObserver = new IntersectionObserver((entries, observer) => {
                entries.forEach(entry => {
                    if (entry.isIntersecting) {
                        const img = entry.target;
                        if (img.dataset.src) {
                            img.src = img.dataset.src;
                            img.removeAttribute('data-src');
                        }
                        observer.unobserve(img);
                    }
                });
            });

            document.querySelectorAll('img[data-src]').forEach(img => {
                imageObserver.observe(img);
            });
        }
    }

    // Dark mode toggle functionality.
    // theme-init.js (loaded in <head>) applies the class to <html> to prevent FOUC.
    // This function syncs that to <body>, updates the icon, and handles toggling.
    function initDarkModeToggle() {
        const toggle = document.getElementById('dark-mode-toggle');

        if (!toggle) {
            return;
        }

        const icon = toggle.querySelector('.dark-mode-icon');

        // Sync from <html> (set by theme-init.js) to <body> for CSS selectors
        if (document.documentElement.classList.contains('dark-mode')) {
            document.body.classList.add('dark-mode');
            if (icon) icon.textContent = '☀️';
        }

        // Toggle dark mode on button click
        toggle.addEventListener('click', function(e) {
            e.preventDefault();

            document.body.classList.toggle('dark-mode');
            document.documentElement.classList.toggle('dark-mode');
            const isDark = document.body.classList.contains('dark-mode');

            // Update icon and save preference
            if (isDark) {
                if (icon) icon.textContent = '☀️';
                localStorage.setItem('theme', 'dark');
            } else {
                if (icon) icon.textContent = '🌙';
                localStorage.setItem('theme', 'light');
            }
        });
    }

    // Smooth scroll for anchor links
    document.addEventListener('click', function(e) {
        if (e.target.tagName === 'A' && e.target.getAttribute('href')?.startsWith('#')) {
            e.preventDefault();
            const target = document.querySelector(e.target.getAttribute('href'));
            if (target) {
                target.scrollIntoView({
                    behavior: 'smooth',
                    block: 'start'
                });
            }
        }
    });

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', init);
    } else {
        init();
    }
})();
