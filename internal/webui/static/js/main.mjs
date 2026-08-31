import {createPageStorage} from "./storage.mjs";
import {setupDialogs} from "./dialogs.mjs";
import {setupForms} from "./forms.mjs";
import {setupNavigation} from "./navigation.mjs";
import {setupPreferences} from "./preferences.mjs";

setupPreferences();
const storage = createPageStorage(document.body.dataset.workspace, window.location.href);
const restored = storage.read();
const status = document.getElementById("live-status");
const announce = (message) => { if (status) status.textContent = message; };
let navigation;
const dialogs = setupDialogs(() => navigation?.source(), announce);
const forms = setupForms({restored, saveState: () => navigation?.saveState(), openDialog: dialogs.open, announce});
navigation = setupNavigation({restored, writeState: storage.write, dialogs, toggleSelection: forms.toggleSelection});
forms.restoreFailures();
if (new URLSearchParams(window.location.search).get("edit") === "1") navigation.editSelected();
