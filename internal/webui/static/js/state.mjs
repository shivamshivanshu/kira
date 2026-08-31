export function initialSelection(items, explicit, previous, samePage) {
  const restored = items.find((item) => item.id === previous && !item.hidden);
  if (samePage && restored) return restored;
  if (explicit) return items.find((item) => item.id === explicit || item.number === explicit) || null;
  return restored || items.find((item) => !item.hidden) || null;
}

export function formSnapshot(controls) {
  return JSON.stringify([...controls]
    .filter((control) => control.name && control.name !== "ids" && !control.disabled && !["hidden", "submit", "button", "reset"].includes(control.type))
    .map((control) => [control.name, control.type, control.type === "checkbox" || control.type === "radio" ? control.checked : control.value]));
}

export function failedFormMatches(form, failure) {
  return !form.hidden && form.action === "/actions/" + failure.action
    && (!failure.id || form.id === failure.id)
    && (!failure.key || form.key === failure.key);
}

export function createUnloadGuard(confirmDiscard) {
  let allowNextUnload = false;
  return {
    allowSubmit(dirtyForms, submittedForm) {
      if ([...dirtyForms].some((form) => form !== submittedForm) && !confirmDiscard()) return false;
      allowNextUnload = true;
      return true;
    },
    shouldWarn(hasDrafts) {
      const allowed = allowNextUnload;
      allowNextUnload = false;
      return hasDrafts && !allowed;
    },
    reset() { allowNextUnload = false; }
  };
}

export const preferenceKey = "kira:preferences:v1";

export function normalizePreferences(value) {
  const input = value && typeof value === "object" ? value : {};
  return {
    theme: ["system", "light", "dark"].includes(input.theme) ? input.theme : "system",
    density: ["comfortable", "compact"].includes(input.density) ? input.density : "comfortable",
    sidebar: ["expanded", "collapsed"].includes(input.sidebar) ? input.sidebar : "expanded"
  };
}

export function resolveTheme(preference, systemDark) {
  if (preference === "light" || preference === "dark") return preference;
  return systemDark ? "dark" : "light";
}

export function normalizePageState(value) {
  const input = value && typeof value === "object" ? value : {};
  const stringList = (v) => Array.isArray(v) ? v.filter((item) => typeof item === "string") : [];
  const offset = (v) => Number.isFinite(v) && v >= 0 ? v : 0;
  return {
    selected: typeof input.selected === "string" ? input.selected : "",
    collapsed: stringList(input.collapsed), checked: stringList(input.checked),
    scroll: offset(input.scroll), boardScroll: offset(input.boardScroll), detailScroll: offset(input.detailScroll),
    url: typeof input.url === "string" ? input.url : ""
  };
}

export function nextActionIndex(index, direction, count) {
  if (!count) return -1;
  if (index < 0) return direction > 0 ? 0 : count - 1;
  return (index + direction + count) % count;
}

export function matchesAction(text, query) {
  return text.toLocaleLowerCase().includes(query.trim().toLocaleLowerCase());
}
