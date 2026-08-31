import {matchesAction, nextActionIndex} from "./state.mjs";

export function setupDialogs(selectedSource, announce) {
  const create = document.getElementById("create-dialog");
  const fallback = document.getElementById("create-item");
  const formHost = document.getElementById("create-form-host");
  const palette = document.getElementById("command-palette");
  const openers = new WeakMap();

  function open(id) {
    const dialog = document.getElementById(id);
    const active = document.querySelector("dialog[open]");
    if (!dialog || typeof dialog.showModal !== "function" || (active && active !== dialog)) return false;
    if (dialog === create) document.getElementById("create-dialog-content")?.append(formHost);
    if (!dialog.open) {
      openers.set(dialog, document.activeElement);
      dialog.showModal();
    }
    dialog.querySelector("input:not([type=hidden]), textarea, .copy-option, .palette-action")?.focus();
    return true;
  }

  document.querySelectorAll("dialog").forEach((dialog) => {
    dialog.addEventListener("close", () => {
      if (dialog === create && formHost) fallback?.append(formHost);
      const opener = openers.get(dialog);
      if (!document.querySelector("dialog[open]") && opener?.isConnected) opener.focus();
    });
  });
  if (create && typeof create.showModal === "function" && fallback) fallback.hidden = true;

  function openCopy(source = selectedSource()) {
    const template = source?.querySelector(".item-copy");
    if (!template) return;
    document.getElementById("copy-options")?.replaceChildren(template.content.cloneNode(true));
    open("copy-dialog");
  }

  async function copy(value) {
    try {
      await navigator.clipboard.writeText(value);
      announce("Copied " + value);
    } catch {
      announce("Clipboard unavailable. Select and copy the reference in the copy picker.");
      openCopy();
    }
  }

  function focusCommand() {
    const panel = document.getElementById("command-panel");
    if (panel) panel.open = true;
    document.getElementById("command-line")?.focus();
  }

  function refresh() { window.location.reload(); }

  const paletteActions = {
    "new-item": () => open("create-dialog"),
    jump: () => open("jump-dialog"),
    command: focusCommand,
    refresh,
    shortcuts: () => open("help-dialog")
  };

  document.addEventListener("click", (event) => {
    const trigger = event.target.closest("[data-open]");
    if (trigger && open(trigger.dataset.open)) event.preventDefault();
    event.target.closest("[data-close]")?.closest("dialog")?.close();
    if (event.target.closest("[data-focus=command-line]")) focusCommand();
    const copyButton = event.target.closest("[data-copy]");
    if (copyButton) void copy(copyButton.dataset.copy);
    if (event.target.closest("[data-copy-picker]")) openCopy(event.target.closest("[data-detail-item]"));
    const action = event.target.closest("[data-palette-action]");
    if (action && palette?.contains(action) && paletteActions[action.dataset.paletteAction]) {
      event.preventDefault();
      openers.delete(palette);
      palette.close();
      paletteActions[action.dataset.paletteAction]();
    }
    const notice = event.target.closest("[data-dismiss-notice]")?.closest(".notice");
    if (notice && !notice.classList.contains("error") && !notice.querySelector("form")) notice.remove();
    const example = event.target.closest("[data-query-example]");
    const query = example && document.getElementById("query");
    if (query) {
      event.preventDefault();
      query.value = example.dataset.queryExample;
      query.dispatchEvent(new Event("input", {bubbles: true}));
      query.focus();
    }
  });

  document.addEventListener("keydown", (event) => {
    if (event.key.toLowerCase() !== "k" || !(event.ctrlKey || event.metaKey) || event.altKey || event.shiftKey) return;
    const active = document.querySelector("dialog[open]");
    if (active && active !== palette) return;
    if (open("command-palette")) event.preventDefault();
  });

  setupActionList("jump-dialog", "jump-query", "[data-jump-item]", false);
  setupActionList("command-palette", "palette-query", ".palette-action", true);
  return {open, openCopy, copy, focusCommand, refresh};
}

function setupActionList(dialogID, inputID, selector, wrap) {
  const dialog = document.getElementById(dialogID);
  const input = document.getElementById(inputID);
  if (!dialog || !input) return;
  const actions = [...dialog.querySelectorAll(selector)];
  input.addEventListener("input", () => {
    actions.forEach((action) => { action.hidden = !matchesAction(action.textContent, input.value); });
    const empty = dialog.querySelector("[data-palette-empty], #palette-empty");
    if (empty) empty.hidden = actions.some((action) => !action.hidden);
  });
  dialog.addEventListener("keydown", (event) => {
    if (!["ArrowDown", "ArrowUp", "Enter"].includes(event.key)) return;
    const visible = actions.filter((action) => !action.hidden && !action.disabled);
    if (event.key === "Enter") {
      if (event.target === input) {
        event.preventDefault();
        visible[0]?.click();
      }
      return;
    }
    event.preventDefault();
    const direction = event.key === "ArrowDown" ? 1 : -1;
    const index = visible.indexOf(document.activeElement);
    const next = wrap ? nextActionIndex(index, direction, visible.length) : Math.max(0, Math.min(visible.length - 1, index + direction));
    visible[next]?.focus();
  });
}
