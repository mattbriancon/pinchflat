// Replaces the LiveView client: server-rendered pages use plain navigations
// (links, GET forms, POST forms that redirect back) instead of htmx, and
// Alpine.js as before.

// phx-hook="supress-enter-submission"
document.addEventListener('keypress', (event) => {
  if (event.key === 'Enter' && event.target.closest('[data-suppress-enter]')) {
    event.preventDefault()
  }
})
