(() => {
  let preferences = {};
  try { preferences = JSON.parse(localStorage.getItem("kira:preferences:v1")) || {}; }
  catch { /* Keep system defaults when storage is unavailable. */ }
  const root = document.documentElement;
  const theme = ["system", "light", "dark"].includes(preferences.theme) ? preferences.theme : "system";
  root.dataset.themePreference = theme;
  root.dataset.theme = theme === "system" ? (window.matchMedia?.("(prefers-color-scheme: dark)").matches ? "dark" : "light") : theme;
  root.dataset.density = ["comfortable", "compact"].includes(preferences.density) ? preferences.density : "comfortable";
  root.dataset.sidebar = ["expanded", "collapsed"].includes(preferences.sidebar) ? preferences.sidebar : "expanded";
})();
