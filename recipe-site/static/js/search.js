// Simple vanilla JavaScript search functionality
(function() {
    'use strict';

    let searchIndex = null;
    let searchInput = null;
    let searchResults = null;
    let debounceTimer = null;
    let indexLoading = false;

    // Initialize search when DOM is ready
    function initSearch() {
        searchInput = document.getElementById('search-input');
        searchResults = document.getElementById('search-results');

        if (!searchInput || !searchResults) return;

        // Lazy-load search index on first focus
        searchInput.addEventListener('focus', loadSearchIndex, { once: true });

        // Add event listener
        searchInput.addEventListener('input', handleSearchInput);

        // Close search results when clicking outside
        document.addEventListener('click', function(e) {
            if (!searchInput.contains(e.target) && !searchResults.contains(e.target)) {
                hideSearchResults();
            }
        });
    }

    function loadSearchIndex() {
        if (searchIndex !== null || indexLoading) return;
        indexLoading = true;

        fetch('/index.json')
            .then(response => response.json())
            .then(data => {
                searchIndex = data;
            })
            .catch(error => console.error('Error loading search index:', error))
            .finally(() => {
                indexLoading = false;
            });
    }

    function handleSearchInput(e) {
        const query = e.target.value.trim();

        // Clear previous timer
        if (debounceTimer) {
            clearTimeout(debounceTimer);
        }

        // Debounce search
        debounceTimer = setTimeout(() => {
            if (query.length < 2) {
                hideSearchResults();
                return;
            }

            if (searchIndex === null) {
                // Index still loading, retry after a short delay
                loadSearchIndex();
                debounceTimer = setTimeout(() => performSearch(query), 500);
                return;
            }

            performSearch(query);
        }, 250);
    }

    function performSearch(query) {
        if (!searchIndex) return;

        const queryLower = query.toLowerCase();
        const results = [];

        searchIndex.forEach(recipe => {
            let score = 0;
            let matchDetails = [];

            // Search in title (highest weight)
            if (recipe.title && recipe.title.toLowerCase().includes(queryLower)) {
                score += 10;
                matchDetails.push('title');
            }

            // Search in description
            if (recipe.description && recipe.description.toLowerCase().includes(queryLower)) {
                score += 5;
                matchDetails.push('description');
            }

            // Search in tags
            if (recipe.tags) {
                recipe.tags.forEach(tag => {
                    if (tag.toLowerCase().includes(queryLower)) {
                        score += 3;
                        matchDetails.push('tag: ' + tag);
                    }
                });
            }

            // Search in keywords
            if (recipe.keywords) {
                recipe.keywords.forEach(keyword => {
                    if (keyword.toLowerCase().includes(queryLower)) {
                        score += 3;
                        matchDetails.push('keyword: ' + keyword);
                    }
                });
            }

            // Search in ingredients
            if (recipe.ingredients) {
                recipe.ingredients.forEach(ingredient => {
                    if (ingredient.toLowerCase().includes(queryLower)) {
                        score += 2;
                        matchDetails.push('ingredient');
                    }
                });
            }

            if (score > 0) {
                results.push({
                    recipe: recipe,
                    score: score,
                    matches: [...new Set(matchDetails)]
                });
            }
        });

        // Sort by score (highest first)
        results.sort((a, b) => b.score - a.score);

        // Display results
        displaySearchResults(results.slice(0, 10), query);
    }

    function displaySearchResults(results, query) {
        if (results.length === 0) {
            searchResults.innerHTML = '<div class="search-no-results">No recipes found</div>';
            searchResults.classList.add('active');
            return;
        }

        let html = '<div class="search-results-list">';

        results.forEach(result => {
            const recipe = result.recipe;

            html += `
                <a href="${recipe.url}" class="search-result-item">
                    <div class="search-result-title">${highlightMatch(recipe.title, query)}</div>
                    ${recipe.description ? `<div class="search-result-description">${truncate(recipe.description, 100)}</div>` : ''}
                    <div class="search-result-meta">
                        ${recipe.tags && recipe.tags.length > 0 ? `<span class="search-result-tags">${recipe.tags.slice(0, 3).join(', ')}</span>` : ''}
                    </div>
                </a>
            `;
        });

        html += '</div>';
        html += `<div class="search-results-footer">${results.length} result${results.length !== 1 ? 's' : ''} found</div>`;

        searchResults.innerHTML = html;
        searchResults.classList.add('active');
    }

    function highlightMatch(text, query) {
        if (!text) return '';

        const regex = new RegExp(`(${escapeRegex(query)})`, 'gi');
        return text.replace(regex, '<mark>$1</mark>');
    }

    function escapeRegex(string) {
        return string.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
    }

    function truncate(text, length) {
        if (text.length <= length) return text;
        return text.substr(0, length) + '...';
    }

    function hideSearchResults() {
        if (searchResults) {
            searchResults.classList.remove('active');
            searchResults.innerHTML = '';
        }
    }

    // Initialize when DOM is ready
    if (document.readyState === 'loading') {
        document.addEventListener('DOMContentLoaded', initSearch);
    } else {
        initSearch();
    }
})();
