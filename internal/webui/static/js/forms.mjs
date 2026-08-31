import {createUnloadGuard, failedFormMatches, formSnapshot} from "./state.mjs";
import {browserStorage, readJSON, writeJSON} from "./storage.mjs";

export function setupForms({restored, saveState, openDialog, announce}) {
  const checkboxes = [...document.querySelectorAll('input[name="ids"][form="bulk-form"]')];
  const dirtyForms = new Set();
  const failedForms = new Set();
  const baselines = new Map();
  const guard = createUnloadGuard(() => window.confirm("Submitting will discard unsaved changes in another form. Continue?"));
  const all = document.getElementById("select-all");
  const bulk = document.getElementById("bulk-form");
  const countLabel = document.getElementById("selection-count");
  const bulkOperation = document.getElementById("bulk-operation");
  const bulkResolution = document.getElementById("bulk-resolution");
  const bulkValue = document.getElementById("bulk-value");

  function updateSelection() {
    const count = checkboxes.filter((input) => input.checked).length;
    if (!all) return;
    all.checked = count > 0 && count === checkboxes.length;
    all.indeterminate = count > 0 && count < checkboxes.length;
    countLabel.textContent = count ? count + " selected" : "Select items";
    bulk.classList.toggle("has-selection", count > 0);
  }

  function updateBulkFields() {
    if (!bulkOperation) return;
    const operation = bulkOperation.value;
    if (bulkResolution) {
      bulkResolution.hidden = operation !== "move";
      bulkResolution.querySelector("input").disabled = operation !== "move";
    }
    const lists = {move: "workflow-states", assign: "people", "label-add": "known-labels", "label-rm": "known-labels", "board-move": "board-keys", sprint: "sprints"};
    const placeholders = {move: "Target state", assign: "Owner (blank clears)", "label-add": "Label to add", "label-rm": "Label to remove", "board-move": "Target board key", sprint: "Sprint (blank clears)"};
    bulkValue.setAttribute("list", lists[operation]);
    bulkValue.placeholder = placeholders[operation];
  }

  if (!document.body.dataset.failedAction && !document.querySelector(".bulk-results")) {
    checkboxes.forEach((input) => { input.checked ||= restored.checked.includes(input.value); });
  }
  updateSelection();
  updateBulkFields();
  bulkOperation?.addEventListener("change", updateBulkFields);
  checkboxes.forEach((input) => input.addEventListener("change", updateSelection));
  all?.addEventListener("change", () => {
    checkboxes.forEach((input) => { input.checked = all.checked; });
    updateSelection();
  });
  document.querySelector("[data-clear-selection]")?.addEventListener("click", () => {
    checkboxes.forEach((input) => { input.checked = false; });
    updateSelection();
  });
  document.querySelectorAll("form").forEach((form) => baselines.set(form, formSnapshot(form.elements)));

  function recordChange(event) {
    const form = event.target.form;
    if (!form || form.method !== "post" || form === bulk) return;
    if (failedForms.has(form) || formSnapshot(form.elements) !== baselines.get(form)) dirtyForms.add(form);
    else dirtyForms.delete(form);
  }

  document.addEventListener("input", recordChange);
  document.addEventListener("change", recordChange);
  document.addEventListener("submit", (event) => {
    if (event.target === bulk && !checkboxes.some((input) => input.checked)) {
      event.preventDefault();
      announce("Select at least one item for the bulk action.");
      return;
    }
    if (!guard.allowSubmit(dirtyForms, event.target)) {
      event.preventDefault();
      return;
    }
    const button = event.submitter;
    if (button) {
      button.dataset.label = button.textContent;
      button.disabled = true;
      button.textContent = "Working…";
    }
    saveState();
  });
  window.addEventListener("beforeunload", (event) => {
    saveState();
    if (guard.shouldWarn(dirtyForms.size > 0)) {
      event.preventDefault();
      event.returnValue = "";
    }
  });
  window.addEventListener("pagehide", saveState);
  window.addEventListener("pageshow", () => {
    guard.reset();
    document.querySelectorAll("button:disabled[data-label]").forEach((button) => {
      button.disabled = false;
      button.textContent = button.dataset.label;
    });
  });

  const refresh = document.getElementById("auto-refresh");
  if (refresh) {
    const storage = browserStorage("sessionStorage");
    const refreshKey = "kira:" + document.body.dataset.workspace + ":auto-refresh";
    refresh.checked = readJSON(storage, refreshKey, false) === true;
    refresh.addEventListener("change", () => writeJSON(storage, refreshKey, refresh.checked));
    window.setInterval(() => {
      if (!refresh.checked || document.hidden || checkboxes.some((input) => input.checked) || document.querySelector("dialog[open]") || document.activeElement?.closest("input, textarea, select, [contenteditable=true]")) return;
      const changed = dirtyForms.size > 0 || [...baselines].some(([form, baseline]) => formSnapshot(form.elements) !== baseline);
      if (!changed) {
        saveState();
        window.location.reload();
      }
    }, 30000);
  }

  function restoreFailures() {
    const body = document.body;
    const failedAction = body.dataset.failedAction;
    if (!failedAction) return;
    const failure = {action: failedAction, id: body.dataset.failedId, key: body.dataset.failedKey};
    for (const form of baselines.keys()) {
      if (!failedFormMatches({action: form.getAttribute("action"), hidden: form.hidden,
        id: form.elements.namedItem("id")?.value, key: form.elements.namedItem("key")?.value}, failure)) continue;
      if (failedAction !== "bulk") {
        failedForms.add(form);
        dirtyForms.add(form);
      }
      for (let node = form.parentElement; node; node = node.parentElement) {
        if (node.tagName === "DETAILS") node.open = true;
      }
    }
    if (failedAction === "create") openDialog("create-dialog");
  }

  function toggleSelection(row) {
    const input = row?.querySelector('input[name="ids"]');
    if (!input) return;
    input.checked = !input.checked;
    updateSelection();
  }

  return {restoreFailures, toggleSelection};
}
