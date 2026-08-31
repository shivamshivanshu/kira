import {normalizePageState} from "./state.mjs";

export function browserStorage(name, scope = globalThis) {
  try { return scope[name] || null; }
  catch { return null; }
}

export function readJSON(storage, key, fallback) {
  try { return JSON.parse(storage?.getItem(key) ?? "null") ?? fallback; }
  catch { return fallback; }
}

export function writeJSON(storage, key, value) {
  try {
    if (!storage) return false;
    storage.setItem(key, JSON.stringify(value));
    return true;
  } catch { return false; }
}

export function pageStateKey(workspace, href) {
  const url = new URL(href);
  ["item", "commit", "edit", "_notice"].forEach((key) => url.searchParams.delete(key));
  return "kira:" + workspace + ":" + url.search;
}

export function createPageStorage(workspace, href) {
  const storage = browserStorage("sessionStorage");
  const key = pageStateKey(workspace, href);
  return {
    read: () => normalizePageState(readJSON(storage, key, {})),
    write: (value) => writeJSON(storage, key, value)
  };
}
