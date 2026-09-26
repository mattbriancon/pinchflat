// Replaces the LiveView client. Server-rendered pages use htmx for the
// interactions LiveView used to handle (table paging/sorting/search, toggles,
// the apprise test button), and Alpine.js as before.

// Progress bar during htmx requests (was phx:page-loading-start/stop).
topbar.config({ barColors: { 0: '#29d' }, shadowColor: 'rgba(0, 0, 0, .3)' })
document.addEventListener('htmx:beforeRequest', () => topbar.show(300))
document.addEventListener('htmx:afterRequest', () => topbar.hide())

// Keep Alpine state on elements htmx swaps (was LiveView's onBeforeElUpdated).
document.addEventListener('htmx:afterSwap', (event) => {
  if (window.Alpine) window.Alpine.initTree(event.detail.elt)
})

// phx-hook="supress-enter-submission"
document.addEventListener('keypress', (event) => {
  if (event.key === 'Enter' && event.target.closest('[data-suppress-enter]')) {
    event.preventDefault()
  }
})
