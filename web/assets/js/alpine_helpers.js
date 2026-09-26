window.copyTextToClipboard = async (text) => {
  // Navigator clipboard api needs a secure context (https)
  if (navigator.clipboard && window.isSecureContext) {
    await navigator.clipboard.writeText(text)
  } else {
    const textArea = document.createElement('textarea')
    textArea.value = text
    // Move textarea out of the viewport so it's not visible
    textArea.style.position = 'absolute'
    textArea.style.left = '-999999px'

    document.body.prepend(textArea)
    textArea.select()

    try {
      document.execCommand('copy')
    } catch (error) {
      console.error(error)
    } finally {
      textArea.remove()
    }
  }
}

window.copyWithCallbacks = async (text, onCopy, onAfterDelay, delay = 4000) => {
  await window.copyTextToClipboard(text)
  onCopy()
  setTimeout(onAfterDelay, delay)
}

window.markVersionAsSeen = (versionString) => {
  localStorage.setItem('seenVersion', versionString)
}

window.isVersionSeen = (versionString) => {
  return localStorage.getItem('seenVersion') === versionString
}

// Replaces Phoenix.LiveView.JS.show/hide(to: "#id") for CoreModal, which has
// no LiveView socket to push a show/hide command over. The modal's own
// x-data listens for these on window and flips its `open` flag.
window.showModal = (id) => {
  window.dispatchEvent(new CustomEvent('pf-show-modal', { detail: id }))
  document.body.classList.add('overflow-hidden')
}

window.hideModal = (id) => {
  window.dispatchEvent(new CustomEvent('pf-hide-modal', { detail: id }))
  document.body.classList.remove('overflow-hidden')
}

window.dispatchFor = (elementOrId, eventName, detail = {}) => {
  const element =
    typeof elementOrId === 'string' ? document.getElementById(elementOrId) : elementOrId

  // This is needed to ensure the DOM has updated before dispatching the event.
  // Doing so ensures that the latest DOM state is what's sent to the server
  setTimeout(() => {
    element.dispatchEvent(new Event(eventName, { bubbles: true, detail }))
  }, 0)
}
