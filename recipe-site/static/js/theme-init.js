// Loaded synchronously in <head> to prevent dark mode flash (FOUC).
// Must run before body renders. Kept minimal for performance.
if (localStorage.getItem('theme') === 'dark') {
    document.documentElement.classList.add('dark-mode');
}
