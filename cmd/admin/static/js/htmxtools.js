document.body.addEventListener('htmx:configRequest', (event) => {
  const csrfToken = document.querySelector('meta[name="csrf-token"]').content
  event.detail.headers['X-CSRF-Token'] = csrfToken
})
