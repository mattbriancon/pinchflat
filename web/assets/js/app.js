// Pages are server-rendered (STRATEGY.md §11): links, GET forms and POST
// forms that redirect back. This file is the only client script. It uses
// event delegation and data-* attributes set by the templates:
//
//   data-copy="text"        click copies text; [data-copied] children show
//                           for 4s and [data-copy-idle] children hide
//   data-dismiss            click removes the closest [role=alert]
//   data-confirm="msg"      on a <form>: confirm() before submitting
//   data-submit-on-change   on an input: submit its form when it changes
//   data-localize           text is a number, shown in the viewer's locale
//   details.dropdown        closes when clicking outside it
//   data-tabs / data-tab / data-tab-panel   tabs, selected by #tab-<id>
//   data-advanced-toggle / data-advanced-only / data-advanced-label
//                           advanced mode, kept in localStorage
//   data-presets='{"preset": value}'   set by the media profile preset button
//   data-preset-select / data-preset-load   the preset picker and its button
//   data-hide-value + data-hide-target      hide target while value matches
//   data-disables / data-sets-value         checkbox disables/fills a field
//   data-fills / data-fallback              select copies its value to a field
//   data-placeholders / data-profile-select / data-load-template
//                           output path placeholder from the chosen profile
;(() => {
  const $ = (sel, root = document) => root.querySelector(sel)
  const $$ = (sel, root = document) => Array.from(root.querySelectorAll(sel))
  const store = {
    get: (k) => {
      try {
        return localStorage.getItem(k)
      } catch {
        return null
      }
    },
    set: (k, v) => {
      try {
        localStorage.setItem(k, v)
      } catch {}
    },
  }

  // --- copy to clipboard ---------------------------------------------------
  const copyText = async (text) => {
    // The clipboard API needs a secure context (https)
    if (navigator.clipboard && window.isSecureContext) {
      await navigator.clipboard.writeText(text)
      return
    }
    const area = document.createElement('textarea')
    area.value = text
    area.style.cssText = 'position:absolute;left:-999999px'
    document.body.prepend(area)
    area.select()
    try {
      document.execCommand('copy')
    } finally {
      area.remove()
    }
  }

  const setCopied = (el, copied) => {
    $$('[data-copied]', el).forEach((n) => (n.hidden = !copied))
    $$('[data-copy-idle]', el).forEach((n) => (n.hidden = copied))
  }

  // --- tabs ----------------------------------------------------------------
  const ACTIVE = ['text-white', 'border-meta-5']
  const INACTIVE = ['border-transparent']

  // An empty hash means the default tab, a #tab-<id> hash picks that tab and
  // any other hash leaves the current tab alone.
  const tabFromHash = (current, fallback) => {
    const hash = location.hash
    if (hash === '' || hash === '#') return fallback
    if (hash.startsWith('#tab-')) return hash.slice('#tab-'.length)
    return current
  }

  const showTab = (box, name) => {
    box.dataset.current = name
    $$('[data-tab]', box).forEach((a) => {
      const on = a.dataset.tab === name
      a.classList.remove(...(on ? INACTIVE : ACTIVE))
      a.classList.add(...(on ? ACTIVE : INACTIVE))
    })
    $$('[data-tab-panel]', box).forEach((p) => (p.hidden = p.dataset.tabPanel !== name))
  }

  const syncTabs = () =>
    $$('[data-tabs]').forEach((box) =>
      showTab(box, tabFromHash(box.dataset.current || box.dataset.tabs, box.dataset.tabs)),
    )

  // --- advanced mode -------------------------------------------------------
  const advanced = () => store.get('advancedMode') === 'true'

  const renderAdvanced = () => {
    const on = advanced()
    $$('[data-advanced-only]').forEach((n) => (n.hidden = !on))
    $$('[data-advanced-label]').forEach((n) => (n.textContent = on ? 'Advanced' : 'Standard'))
  }

  // --- conditional fields --------------------------------------------------
  const renderHidden = () =>
    $$('[data-hide-target]').forEach((el) => {
      const target = $(el.dataset.hideTarget)
      if (target) target.hidden = el.value === el.dataset.hideValue
    })

  const renderDisables = () =>
    $$('[data-disables]').forEach((el) => {
      const target = $(el.dataset.disables)
      if (target) target.disabled = el.checked
    })

  const renderPlaceholders = () =>
    $$('[data-placeholders]').forEach((el) => {
      const profile = $(el.dataset.profileSelect)
      const map = JSON.parse(el.dataset.placeholders)
      el.placeholder = (profile && map[profile.value]) || ''
    })

  // --- media profile presets -----------------------------------------------
  const changed = (el) => el.dispatchEvent(new Event('change', { bubbles: true }))

  const applyPreset = (form, name) => {
    $$('[data-presets]', form).forEach((el) => {
      const value = JSON.parse(el.dataset.presets)[name]
      if (el.type === 'checkbox') el.checked = !!value
      else el.value = value ?? ''
      changed(el)
    })
  }

  const renderPresetButton = (select) => {
    const btn = $('[data-preset-load]', select.form)
    if (!btn) return
    btn.disabled = !select.value
    $$('[data-preset-label]', btn).forEach((n) => (n.textContent = select.value ? 'Load' : 'Select'))
  }

  // --- events --------------------------------------------------------------
  document.addEventListener('click', async (e) => {
    const t = e.target.closest ? e.target : e.target.parentElement
    if (!t) return

    // Close open dropdown menus when clicking anywhere outside them.
    $$('details.dropdown[open]').forEach((d) => d.contains(t) || d.removeAttribute('open'))

    const dismiss = t.closest('[data-dismiss]')
    if (dismiss) dismiss.closest('[role=alert]')?.remove()

    if (t.closest('[data-advanced-toggle]')) {
      store.set('advancedMode', String(!advanced()))
      renderAdvanced()
    }

    const version = t.closest('[data-version]')
    if (version) {
      store.set('seenVersion', version.dataset.version)
      $$('[data-new-badge]').forEach((n) => (n.hidden = true))
    }

    const load = t.closest('[data-preset-load]')
    if (load && load.form) {
      const select = $('[data-preset-select]', load.form)
      if (select && select.value) {
        applyPreset(load.form, select.value)
        select.value = ''
        renderPresetButton(select)
      }
    }

    if (t.closest('[data-load-template]')) {
      const input = $('[data-placeholders]')
      const profile = input && $(input.dataset.profileSelect)
      if (input && profile) input.value = JSON.parse(input.dataset.placeholders)[profile.value] || ''
    }

    const copy = t.closest('[data-copy]')
    if (copy) {
      e.preventDefault()
      try {
        await copyText(copy.dataset.copy)
      } catch (err) {
        console.error(err)
        return
      }
      setCopied(copy, true)
      setTimeout(() => setCopied(copy, false), 4000)
    }
  })

  document.addEventListener('change', (e) => {
    const el = e.target

    if (el.matches('[data-submit-on-change]')) el.form.requestSubmit()
    if (el.matches('[data-preset-select]')) renderPresetButton(el)
    if (el.matches('[data-hide-target]')) renderHidden()
    if (el.matches('[data-disables]')) {
      renderDisables()
      const target = $(el.dataset.disables)
      if (el.checked && el.dataset.setsValue && target) target.value = el.dataset.setsValue
    }
    if (el.matches('[data-fills]')) {
      const target = $(el.dataset.fills)
      if (target) target.value = el.value || el.dataset.fallback || ''
    }
    if ($('[data-placeholders]')) renderPlaceholders()
  })

  document.addEventListener('submit', (e) => {
    const msg = e.target.dataset && e.target.dataset.confirm
    if (msg && !confirm(msg)) e.preventDefault()
  })

  window.addEventListener('hashchange', syncTabs)

  const init = () => {
    $$('[data-localize]').forEach((n) => {
      const v = Number(n.textContent)
      if (Number.isFinite(v)) n.textContent = new Intl.NumberFormat().format(v)
    })
    const badge = $('[data-new-badge]')
    if (badge && store.get('seenVersion') !== badge.dataset.newBadge) badge.hidden = false
    $$('[data-preset-select]').forEach(renderPresetButton)
    syncTabs()
    renderAdvanced()
    renderHidden()
    renderDisables()
    renderPlaceholders()
  }

  if (document.readyState === 'loading') document.addEventListener('DOMContentLoaded', init)
  else init()
})()
