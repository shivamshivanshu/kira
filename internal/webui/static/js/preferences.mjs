import {normalizePreferences, preferenceKey, resolveTheme} from "./state.mjs";
import {browserStorage, readJSON, writeJSON} from "./storage.mjs";

export function setupPreferences(doc = document, win = window) {
  const storage = browserStorage("localStorage", win);
  const media = win.matchMedia?.("(prefers-color-scheme: dark)");
  const theme = doc.getElementById("theme-preference");
  const density = doc.getElementById("density-preference");
  const sidebar = doc.getElementById("sidebar-toggle");
  let current = normalizePreferences(readJSON(storage, preferenceKey, null));

  function apply() {
    const root = doc.documentElement;
    root.dataset.themePreference = current.theme;
    root.dataset.theme = resolveTheme(current.theme, media?.matches);
    root.dataset.density = current.density;
    root.dataset.sidebar = current.sidebar;
    if (theme) theme.value = current.theme;
    if (density) density.value = current.density;
    if (sidebar) {
      const expanded = current.sidebar === "expanded";
      sidebar.setAttribute("aria-expanded", String(expanded));
      sidebar.setAttribute("aria-controls", "workspace-sidebar");
      sidebar.setAttribute("aria-label", expanded ? "Collapse sidebar" : "Expand sidebar");
      sidebar.title = expanded ? "Collapse sidebar" : "Expand sidebar";
    }
  }

  function update(change) {
    current = normalizePreferences({...current, ...change});
    apply();
    writeJSON(storage, preferenceKey, current);
  }

  theme?.addEventListener("change", () => update({theme: theme.value}));
  density?.addEventListener("change", () => update({density: density.value}));
  sidebar?.addEventListener("click", () => update({sidebar: current.sidebar === "expanded" ? "collapsed" : "expanded"}));
  const onSystemChange = () => { if (current.theme === "system") apply(); };
  if (media?.addEventListener) media.addEventListener("change", onSystemChange);
  else media?.addListener?.(onSystemChange);
  win.addEventListener("storage", (event) => {
    if (event.storageArea && event.storageArea !== storage) return;
    if (event.key !== preferenceKey && event.key !== null) return;
    current = normalizePreferences(readJSON(storage, preferenceKey, null));
    apply();
  });
  apply();
}
