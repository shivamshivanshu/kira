import {matchesAction, nextActionIndex, pickerChoices} from "./state.mjs";

export function setupDialogs(selectedSource, announce) {
  const create = document.getElementById("create-dialog");
  const fallback = document.getElementById("create-item");
  const formHost = document.getElementById("create-form-host");
  const palette = document.getElementById("command-palette");
  const openers = new WeakMap();
  const refreshJump = setupJumpPicker();

  function open(id) {
    const dialog = document.getElementById(id);
    const active = document.querySelector("dialog[open]");
    if (!dialog || typeof dialog.showModal !== "function" || (active && active !== dialog)) return false;
    if (id === "jump-dialog") refreshJump();
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
    const template = document.getElementById("item-copy");
    if (!template || !source?.dataset.copyValues) return;
    const values = JSON.parse(source.dataset.copyValues);
    const content = template.content.cloneNode(true);
    content.querySelectorAll(".copy-option").forEach((button, index) => {
      const value = values[index];
      if (value === undefined) { button.remove(); return; }
      button.dataset.copy = value;
      button.querySelector("code").textContent = value;
    });
    document.getElementById("copy-options")?.replaceChildren(content);
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

  setupActionList("jump-dialog", "jump-query", "[data-jump-item]", {wrap: false, filter: false});
  setupActionList("command-palette", "palette-query", ".palette-action", {wrap: true, filter: true});
  return {open, openCopy, copy, focusCommand, refresh};
}

function setupActionList(dialogID, inputID, selector, {wrap, filter}) {
  const dialog = document.getElementById(dialogID);
  const input = document.getElementById(inputID);
  if (!dialog || !input) return;
  const actions = () => [...dialog.querySelectorAll(selector)];
  if (filter) input.addEventListener("input", () => {
    actions().forEach((action) => { action.hidden = !matchesAction(action.textContent, input.value); });
    const empty = dialog.querySelector("[data-palette-empty], #palette-empty");
    if (empty) empty.hidden = actions().some((action) => !action.hidden);
  });
  dialog.addEventListener("keydown", (event) => {
    if (!["ArrowDown", "ArrowUp", "Enter"].includes(event.key)) return;
    const visible = actions().filter((action) => !action.hidden && !action.disabled);
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

function setupJumpPicker() {
  const input = document.getElementById("jump-query");
  const data = document.getElementById("jump-items");
  const results = document.querySelector("#jump-dialog .jump-results");
  const count = document.getElementById("jump-count");
  if (!input || !data || !results) return () => {};
  let items;
  function refresh() {
    items ??= JSON.parse(data.textContent).map((item) => ({...item,
      search: `${item.number} ${item.title} ${item.state.replaceAll("_", " ")}`.toLocaleLowerCase()}));
    const {choices, total} = pickerChoices(items, input.value);
    const fragment = document.createDocumentFragment();
    for (const item of choices) {
      const link = document.createElement("a");
      const url = new URL(results.dataset.itemUrl, window.location.href);
      url.searchParams.set("item", item.id);
      link.href = url.pathname + url.search;
      link.dataset.jumpItem = "";
      const number = document.createElement("code");
      number.textContent = item.number;
      const title = document.createElement("span");
      title.textContent = item.title;
      const state = document.createElement("span");
      state.className = `state category-${item.category}`;
      state.textContent = item.state.replaceAll("_", " ");
      link.append(number, title, state);
      fragment.append(link);
    }
    results.replaceChildren(fragment);
    if (count) count.textContent = total > 50 ? `Showing 50 of ${total} matches. Refine your search to find an item.` : `${total} matching items`;
  }
  input.addEventListener("input", refresh);
  return refresh;
}
