import test from "node:test";
import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import {runInNewContext} from "node:vm";
import {normalizePageState, normalizePreferences, preferenceKey, resolveTheme, nextActionIndex, matchesAction} from "../static/js/state.mjs";
import {browserStorage, pageStateKey, readJSON, writeJSON} from "../static/js/storage.mjs";
import {setupPreferences} from "../static/js/preferences.mjs";
import {setupDialogs} from "../static/js/dialogs.mjs";

function memoryStorage(value = null) {
  const values = new Map(value ? [[preferenceKey, JSON.stringify(value)]] : []);
  return {
    getItem: (key) => values.get(key) ?? null,
    setItem: (key, value) => values.set(key, value),
    clear: () => values.clear()
  };
}

function eventNode() {
  const listeners = new Map();
  return {
    value: "", attributes: {},
    addEventListener(name, listener) { listeners.set(name, listener); },
    setAttribute(name, value) { this.attributes[name] = value; },
    emit(name, properties = {}) { listeners.get(name)?.({target: this, ...properties}); }
  };
}

function preferenceFixture(value = null, dark = false, storage = memoryStorage(value)) {
  const theme = eventNode(), density = eventNode(), sidebar = eventNode();
  const nodes = {"theme-preference": theme, "density-preference": density, "sidebar-toggle": sidebar};
  const doc = {documentElement: {dataset: {}}, getElementById: (id) => nodes[id]};
  const media = Object.assign(eventNode(), {matches: dark});
  const win = Object.assign(eventNode(), {localStorage: storage, matchMedia: () => media});
  return {doc, win, media, theme, density, sidebar, storage, root: doc.documentElement.dataset};
}

test("invalid or partial preferences fall back without losing valid choices", () => {
  const defaults = {theme: "system", density: "comfortable", sidebar: "expanded"};
  for (const value of [null, "dark", 42, [], {theme: "unknown", density: {}, sidebar: false}]) {
    assert.deepEqual(normalizePreferences(value), defaults);
  }
  assert.deepEqual(normalizePreferences({theme: "dark", density: "compact"}), {theme: "dark", density: "compact", sidebar: "expanded"});
  assert.equal(resolveTheme("system", true), "dark");
  assert.equal(resolveTheme("system", false), "light");
  assert.equal(resolveTheme("light", true), "light");
});

test("browser storage denial, invalid JSON, and quota errors do not escape", () => {
  const scope = Object.defineProperty({}, "localStorage", {get() { throw new Error("denied"); }});
  assert.equal(browserStorage("localStorage", scope), null);
  const unavailable = {getItem() { throw new Error("denied"); }, setItem() { throw new Error("quota"); }};
  assert.deepEqual(readJSON(unavailable, preferenceKey, {}), {});
  assert.equal(writeJSON(unavailable, preferenceKey, {theme: "dark"}), false);
  assert.equal(writeJSON(null, preferenceKey, {}), false);
  assert.equal(readJSON({getItem: () => "{broken"}, preferenceKey, "fallback"), "fallback");
  const storage = memoryStorage();
  assert.equal(writeJSON(storage, preferenceKey, {theme: "dark"}), true);
  assert.deepEqual(readJSON(storage, preferenceKey, {}), {theme: "dark"});
});

test("page state keys isolate workspaces and filters while discarding transient navigation", () => {
  const href = "http://localhost:7432/?view=board&q=owner%3Dme&item=A&commit=SHA&edit=1&_notice=once";
  assert.equal(pageStateKey("repo", href), "kira:repo:?view=board&q=owner%3Dme");
  assert.notEqual(pageStateKey("repo", href), pageStateKey("other", href));
  assert.notEqual(pageStateKey("repo", href), pageStateKey("repo", "http://localhost:7432/?view=tree"));
});

test("corrupt saved page state cannot break selection or scrolling", () => {
  const state = normalizePageState({selected: {}, checked: {}, collapsed: ["epic", null], scroll: -4, boardScroll: Infinity, detailScroll: 32});
  assert.deepEqual(state.checked, []);
  assert.deepEqual(state.collapsed, ["epic"]);
  assert.equal(state.selected, "");
  assert.equal(state.scroll, 0);
  assert.equal(state.boardScroll, 0);
  assert.equal(state.detailScroll, 32);
});

test("system theme follows media changes and explicit theme overrides them", () => {
  const f = preferenceFixture(null, false);
  setupPreferences(f.doc, f.win);
  assert.equal(f.root.theme, "light");
  f.media.matches = true;
  f.media.emit("change");
  assert.equal(f.root.theme, "dark");
  f.theme.value = "light";
  f.theme.emit("change");
  f.media.emit("change");
  assert.equal(f.root.theme, "light");
  f.theme.value = "system";
  f.theme.emit("change");
  assert.equal(f.root.theme, "dark");
  assert.equal(readJSON(f.storage, preferenceKey, {}).theme, "system");
});

test("density and sidebar preferences persist and retain accessible toggle state", () => {
  const f = preferenceFixture({theme: "dark", density: "compact", sidebar: "collapsed"});
  setupPreferences(f.doc, f.win);
  assert.equal(f.root.density, "compact");
  assert.equal(f.density.value, "compact");
  assert.equal(f.sidebar.attributes["aria-expanded"], "false");
  assert.equal(f.sidebar.attributes["aria-controls"], "workspace-sidebar");
  f.sidebar.emit("click");
  f.density.value = "comfortable";
  f.density.emit("change");
  assert.equal(f.root.sidebar, "expanded");
  assert.equal(f.sidebar.attributes["aria-expanded"], "true");
  assert.deepEqual(readJSON(f.storage, preferenceKey, {}), {theme: "dark", density: "comfortable", sidebar: "expanded"});
});

test("cross-tab updates and clearing storage synchronize appearance controls", () => {
  const f = preferenceFixture(null, true);
  setupPreferences(f.doc, f.win);
  writeJSON(f.storage, preferenceKey, {theme: "light", density: "compact", sidebar: "collapsed"});
  f.win.emit("storage", {key: preferenceKey, storageArea: f.storage});
  assert.equal(f.root.theme, "light");
  assert.equal(f.theme.value, "light");
  assert.equal(f.density.value, "compact");
  assert.equal(f.sidebar.attributes["aria-expanded"], "false");
  writeJSON(f.storage, preferenceKey, {theme: "dark"});
  f.win.emit("storage", {key: "unrelated", storageArea: f.storage});
  assert.equal(f.root.theme, "light");
  f.win.emit("storage", {key: preferenceKey, storageArea: memoryStorage()});
  assert.equal(f.root.theme, "light");
  f.storage.clear();
  f.win.emit("storage", {key: null, storageArea: f.storage});
  assert.equal(f.root.themePreference, "system");
  assert.equal(f.root.theme, "dark");
  assert.equal(f.density.value, "comfortable");
});

test("appearance controls still work when local storage is blocked", () => {
  const storage = {getItem() { throw new Error("blocked"); }, setItem() { throw new Error("blocked"); }};
  const f = preferenceFixture(null, false, storage);
  setupPreferences(f.doc, f.win);
  f.theme.value = "dark";
  assert.doesNotThrow(() => f.theme.emit("change"));
  assert.equal(f.root.theme, "dark");
  f.sidebar.emit("click");
  assert.equal(f.root.sidebar, "collapsed");
});

const appearanceScript = readFileSync(new URL("../static/js/appearance.js", import.meta.url), "utf8");

test("before-paint bootstrap and runtime agree for supported preferences", () => {
  for (const theme of ["system", "light", "dark"]) {
    for (const dark of [false, true]) {
      const f = preferenceFixture({theme, density: "compact", sidebar: "collapsed"}, dark);
      runInNewContext(appearanceScript, {document: f.doc, window: f.win, localStorage: f.storage});
      const before = {...f.root};
      setupPreferences(f.doc, f.win);
      assert.deepEqual(f.root, before);
      assert.equal(f.root.theme, resolveTheme(theme, dark));
    }
  }
});

test("before-paint bootstrap follows the system even when storage is inaccessible", () => {
  const f = preferenceFixture(null, true);
  const storage = {getItem() { throw new Error("blocked"); }};
  runInNewContext(appearanceScript, {document: f.doc, window: f.win, localStorage: storage});
  assert.equal(f.root.theme, "dark");
  assert.equal(f.root.themePreference, "system");
  assert.equal(f.root.density, "comfortable");
  assert.equal(f.root.sidebar, "expanded");
});

test("palette arrows enter the list from search and wrap without invalid empty targets", () => {
  assert.equal(nextActionIndex(-1, 1, 3), 0);
  assert.equal(nextActionIndex(-1, -1, 3), 2);
  assert.equal(nextActionIndex(2, 1, 3), 0);
  assert.equal(nextActionIndex(0, -1, 3), 2);
  assert.equal(nextActionIndex(-1, 1, 0), -1);
});

test("palette filtering is case insensitive and handles empty or unmatched searches", () => {
  assert.equal(matchesAction("Create a new item", "  NEW ITEM  "), true);
  assert.equal(matchesAction("Project settings", "settings"), true);
  assert.equal(matchesAction("Refresh dashboard", ""), true);
  assert.equal(matchesAction("Create a new item", "board"), false);
});

function paletteFixture() {
  const doc = eventNode();
  const focusable = () => Object.assign(eventNode(), {isConnected: true, focus() { doc.activeElement = this; }});
  const opener = focusable(), search = focusable(), command = focusable();
  const panel = {open: false};
  const pending = [];
  const action = {dataset: {paletteAction: "command"}};
  const palette = Object.assign(eventNode(), {
    open: false,
    showModal() { this.open = true; },
    close() { this.open = false; pending.push(() => this.emit("close")); },
    contains: (node) => node === action,
    querySelector: () => search,
    querySelectorAll: () => []
  });
  const ids = {"command-palette": palette, "palette-query": search, "command-panel": panel, "command-line": command};
  doc.getElementById = (id) => ids[id] || null;
  doc.querySelector = () => palette.open ? palette : null;
  doc.querySelectorAll = () => [palette];
  doc.activeElement = opener;
  const click = (selector, target) => doc.emit("click", {
    target: {closest: (query) => query === selector ? target : null},
    preventDefault() {}
  });
  return {doc, opener, search, command, palette, action, click, flush: () => pending.splice(0).forEach((callback) => callback())};
}

function withDocument(doc, run) {
  const original = globalThis.document;
  globalThis.document = doc;
  try { run(); }
  finally {
    if (original === undefined) delete globalThis.document;
    else globalThis.document = original;
  }
}

test("queued palette close cannot steal focus from its chosen command action", () => {
  const f = paletteFixture();
  withDocument(f.doc, () => {
    const dialogs = setupDialogs(() => null, () => {});
    dialogs.open("command-palette");
    assert.equal(f.doc.activeElement, f.search);
    f.click("[data-palette-action]", f.action);
    assert.equal(f.doc.activeElement, f.command);
    f.flush();
    assert.equal(f.doc.activeElement, f.command);
  });
});

test("normal Escape and close-button paths still return focus to the opener", () => {
  for (const closeButton of [false, true]) {
    const f = paletteFixture();
    withDocument(f.doc, () => {
      const dialogs = setupDialogs(() => null, () => {});
      dialogs.open("command-palette");
      if (closeButton) f.click("[data-close]", {closest: () => f.palette});
      else f.palette.close();
      f.flush();
      assert.equal(f.doc.activeElement, f.opener);
    });
  }
});
