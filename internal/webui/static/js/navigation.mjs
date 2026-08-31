import {initialSelection} from "./state.mjs";

export function setupNavigation({restored, writeState, dialogs, toggleSelection}) {
  const body = document.body;
  const rows = [...document.querySelectorAll("[data-nav-item]")];
  const treeRows = rows.filter((row) => row.hasAttribute("data-depth"));
  const columns = [...document.querySelectorAll(".kanban-column")];
  const checkboxes = [...document.querySelectorAll('input[name="ids"][form="bulk-form"]')];
  const selectedFields = [...document.querySelectorAll('input[name="selected"]')];
  const board = document.querySelector(".kanban");
  const detail = document.querySelector(".detail-pane");
  const detailItem = document.querySelector("[data-detail-item]");
  const collapsed = new Set(restored.collapsed);
  let selected = null;
  let prefix = "";
  let prefixAt = 0;

  const visibleRows = () => rows.filter((row) => !row.hidden);
  const source = () => selected || detailItem;

  function select(row, focus = true) {
    if (!row) return;
    selected?.classList.remove("is-cursor");
    selected = row;
    selected.classList.add("is-cursor");
    selectedFields.forEach((input) => { input.value = row.dataset.item; });
    if (focus) {
      row.focus({preventScroll: true});
      row.scrollIntoView({block: "nearest", inline: "nearest"});
    }
  }

  function saveState() {
    writeState({
      selected: selected?.dataset.item, collapsed: [...collapsed],
      checked: checkboxes.filter((input) => input.checked).map((input) => input.value),
      scroll: window.scrollY, boardScroll: board?.scrollLeft || 0, detailScroll: detail?.scrollTop || 0,
      url: window.location.href
    });
  }

  function updateTree() {
    const ancestors = [];
    for (const row of treeRows) {
      const depth = Number(row.dataset.depth);
      while (ancestors.length && ancestors.at(-1).depth >= depth) ancestors.pop();
      row.hidden = ancestors.some((entry) => entry.collapsed);
      const toggle = row.querySelector(".tree-toggle");
      if (toggle && !toggle.hidden) {
        const isCollapsed = collapsed.has(row.dataset.item);
        toggle.setAttribute("aria-expanded", String(!isCollapsed));
        toggle.setAttribute("aria-label", (isCollapsed ? "Expand " : "Collapse ") + row.dataset.number);
        ancestors.push({depth, collapsed: isCollapsed});
      }
    }
    if (selected?.hidden) select(visibleRows()[0], false);
  }

  function expandAll(expand) {
    treeRows.forEach((row) => {
      if (expand) collapsed.delete(row.dataset.item);
      else {
        const toggle = row.querySelector(".tree-toggle");
        if (toggle && !toggle.hidden) collapsed.add(row.dataset.item);
      }
    });
    updateTree();
    saveState();
  }

  treeRows.forEach((row, index) => {
    const toggle = row.querySelector(".tree-toggle");
    if (!toggle || !treeRows[index + 1] || Number(treeRows[index + 1].dataset.depth) <= Number(row.dataset.depth)) return;
    toggle.hidden = false;
    toggle.addEventListener("click", () => {
      if (collapsed.has(row.dataset.item)) collapsed.delete(row.dataset.item);
      else collapsed.add(row.dataset.item);
      updateTree();
      select(row);
      saveState();
    });
  });
  updateTree();

  document.addEventListener("click", (event) => {
    const row = event.target.closest("[data-nav-item]");
    if (row) select(row, false);
  });
  document.addEventListener("focusin", (event) => {
    const row = event.target.closest("[data-nav-item]");
    if (row) select(row, false);
  });
  document.querySelector("[data-tree-expand]")?.addEventListener("click", () => expandAll(true));
  document.querySelector("[data-tree-collapse]")?.addEventListener("click", () => expandAll(false));

  function vertical(direction) {
    const scope = selected?.closest(".kanban-column");
    const items = scope ? [...scope.querySelectorAll("[data-nav-item]")] : visibleRows();
    const index = items.indexOf(selected);
    select(items[Math.max(0, Math.min(items.length - 1, index + direction))]);
  }

  function horizontal(direction) {
    if (!selected) { select(visibleRows()[0]); return; }
    const column = selected.closest(".kanban-column");
    if (column) {
      const index = columns.indexOf(column);
      const position = [...column.querySelectorAll("[data-nav-item]")].indexOf(selected);
      for (let i = index + direction; i >= 0 && i < columns.length; i += direction) {
        const candidates = [...columns[i].querySelectorAll("[data-nav-item]")];
        if (candidates.length) { select(candidates[Math.min(position, candidates.length - 1)]); break; }
      }
      return;
    }
    const index = treeRows.indexOf(selected);
    const depth = Number(selected.dataset.depth);
    const toggle = selected.querySelector(".tree-toggle");
    if (!toggle) return;
    if (direction > 0) {
      if (!toggle.hidden && collapsed.has(selected.dataset.item)) {
        collapsed.delete(selected.dataset.item);
        updateTree();
      } else if (treeRows[index + 1] && Number(treeRows[index + 1].dataset.depth) > depth) select(treeRows[index + 1]);
    } else if (!toggle.hidden && !collapsed.has(selected.dataset.item)) {
      collapsed.add(selected.dataset.item);
      updateTree();
    } else {
      for (let i = index - 1; i >= 0; i--) {
        if (Number(treeRows[i].dataset.depth) < depth) { select(treeRows[i]); break; }
      }
    }
  }

  function moveBoard(direction) {
    const column = selected?.closest(".kanban-column");
    const target = columns[columns.indexOf(column) + direction];
    const form = document.getElementById("keyboard-move");
    if (!column || !target || !form) return;
    form.elements.id.value = selected.dataset.item;
    form.elements.state.value = target.dataset.state;
    form.requestSubmit();
  }

  function editSelected() {
    const item = source();
    if (detailItem && (!item || item.dataset.item === detailItem.dataset.item)) {
      const editor = detailItem.querySelector(".edit-section");
      editor.open = true;
      editor.querySelector('input[name="title"]')?.focus();
    } else {
      const link = item?.querySelector("[data-item-link]");
      if (link) {
        const url = new URL(link.href);
        url.searchParams.set("edit", "1");
        window.location.assign(url);
      }
    }
  }

  document.addEventListener("keydown", (event) => {
    const editing = event.target.closest("input, textarea, select, [contenteditable=true]");
    if (event.ctrlKey && !event.altKey && !event.metaKey && !editing && ["o", "i"].includes(event.key)) {
      event.preventDefault();
      window.history.go(event.key === "o" ? -1 : 1);
      return;
    }
    if (event.ctrlKey || event.metaKey || event.altKey || editing || document.querySelector("dialog[open]")) return;
    const second = Date.now() - prefixAt < 1000 ? prefix : "";
    prefix = "";
    if (second === "g" && event.key === "g") { event.preventDefault(); select(visibleRows()[0]); return; }
    if (second === "g" && event.key === "p") {
      const url = source()?.dataset.parentUrl;
      if (url) window.location.assign(url);
      return;
    }
    if (second === "z" && ["M", "R"].includes(event.key)) { expandAll(event.key === "R"); return; }
    if (["g", "z"].includes(event.key)) { prefix = event.key; prefixAt = Date.now(); return; }
    if (["1", "2", "3", "4"].includes(event.key)) {
      event.preventDefault();
      document.querySelector('[data-view="' + event.key + '"]')?.click();
    } else if (["j", "k", "ArrowDown", "ArrowUp"].includes(event.key)) {
      event.preventDefault();
      vertical(["j", "ArrowDown"].includes(event.key) ? 1 : -1);
    } else if (["h", "l", "ArrowLeft", "ArrowRight"].includes(event.key)) {
      event.preventDefault();
      horizontal(["l", "ArrowRight"].includes(event.key) ? 1 : -1);
    } else if (["H", "L"].includes(event.key)) {
      event.preventDefault();
      moveBoard(event.key === "L" ? 1 : -1);
    } else if (event.key === "G") {
      event.preventDefault();
      select(visibleRows().at(-1));
    } else if (event.key === "Enter" && !event.target.closest("a, button, summary")) {
      event.preventDefault();
      selected?.querySelector("[data-item-link]")?.click();
    } else if (event.key === " " && selected && !event.target.closest("button, summary")) {
      event.preventDefault();
      toggleSelection(selected);
    } else if (event.key === "y") {
      const value = source()?.querySelector(".item-copy")?.content.querySelector('[data-copy-format="id"]')?.dataset.copy;
      if (value) void dialogs.copy(value);
    } else if (event.key === "Y") dialogs.openCopy();
    else if (event.key === "e") editSelected();
    else if (event.key === ":") { event.preventDefault(); dialogs.focusCommand(); }
    else if (event.key === "/") { event.preventDefault(); document.getElementById("query")?.focus(); }
    else if (event.key === "p") dialogs.open("jump-dialog");
    else if (event.key === "n") dialogs.open("create-dialog");
    else if (event.key === "?") dialogs.open("help-dialog");
    else if (event.key === "r") dialogs.refresh();
    else if (event.key === "Escape") document.querySelector("[data-close-detail]")?.click();
  });

  const initial = initialSelection(rows.map((row) => ({id: row.dataset.item, number: row.dataset.number, hidden: row.hidden})), body.dataset.selected, restored.selected, restored.url === window.location.href);
  const initialRow = initial && rows.find((row) => row.dataset.item === initial.id);
  if (initialRow?.hidden) {
    let depth = Number(initialRow.dataset.depth);
    for (let index = treeRows.indexOf(initialRow) - 1; index >= 0; index--) {
      const ancestor = treeRows[index];
      if (Number(ancestor.dataset.depth) < depth) {
        collapsed.delete(ancestor.dataset.item);
        depth = Number(ancestor.dataset.depth);
      }
    }
    updateTree();
  }
  select(initialRow, false);
  if (restored.url === window.location.href) {
    window.requestAnimationFrame(() => {
      window.scrollTo(0, restored.scroll);
      if (board) board.scrollLeft = restored.boardScroll;
      if (detail) detail.scrollTop = restored.detailScroll;
    });
  }
  return {source, saveState, editSelected};
}
