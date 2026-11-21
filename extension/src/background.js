// ScamShield AI - background service worker (Manifest V3)

const MENU_ID = 'scamshield-scan-selection'

// Create (or refresh) the right-click entry for selected text.
chrome.runtime.onInstalled.addListener(() => {
  chrome.contextMenus.removeAll(() => {
    chrome.contextMenus.create({
      id: MENU_ID,
      title: 'Scan with ScamShield AI',
      contexts: ['selection'],
    })
  })
})

chrome.contextMenus.onClicked.addListener((info) => {
  if (info.menuItemId !== MENU_ID) return

  const selectedText = (info.selectionText || '').trim()
  if (!selectedText) return

  // Hand the selection over to the popup and let it start the scan itself.
  chrome.storage.local.set(
    {
      pendingScan: selectedText,
      autoScan: true,
    },
    () => {
      chrome.action.openPopup().catch(() => {
        // openPopup() is only available on Chrome 127+; the user can still
        // click the toolbar icon and the text will be waiting for them.
      })
    },
  )
})
