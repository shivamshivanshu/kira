import test from "node:test";
import assert from "node:assert/strict";
import {initialSelection, formSnapshot, failedFormMatches, createUnloadGuard, pickerChoices} from "../static/js/state.mjs";

test("picker limits rendered choices while searching the entire repository", () => {
  const items = Array.from({length: 1000}, (_, i) => ({id: i, search: `web-${i} title`}));
  const all = pickerChoices(items, "");
  assert.equal(all.choices.length, 50);
  assert.equal(all.total, 1000);
  assert.deepEqual(pickerChoices(items, " WEB-999 ").choices, [items[999]]);
  assert.equal(pickerChoices(items, "missing").total, 0);
  assert.deepEqual(pickerChoices([], ""), {choices: [], total: 0});
});

test("an explicitly opened item outside the filter does not select another item", () => {
  const rows = [{id: "visible", number: "WEB-1", hidden: false}];
  assert.equal(initialSelection(rows, "WEB-2", "visible", false), null);
});

test("a newly opened hidden child remains the target so its parents can expand", () => {
  const rows = [
    {id: "parent", number: "WEB-1", hidden: false},
    {id: "child", number: "WEB-2", hidden: true},
  ];
  assert.equal(initialSelection(rows, "WEB-2", "parent", false).id, "child");
});

test("refresh restores an intentionally selected row and navigation honors a new item", () => {
  const rows = [
    {id: "a", number: "WEB-1", hidden: false},
    {id: "b", number: "WEB-2", hidden: false},
  ];
  assert.equal(initialSelection(rows, "WEB-1", "b", true).id, "b");
  assert.equal(initialSelection(rows, "WEB-1", "b", false).id, "a");
});

test("a command error redirect keeps its target instead of the previously open detail", () => {
  const rows = [
    {id: "item-a", number: "WEB-1", hidden: false},
    {id: "item-b", number: "WEB-2", hidden: false},
  ];
  assert.equal(initialSelection(rows, "WEB-2", "item-a", false).id, "item-b");
  assert.equal(initialSelection(rows.slice(0, 1), "WEB-2", "item-a", false), null);
});

test("reverting a field restores its original snapshot", () => {
  const fields = [{name: "title", type: "text", value: "Original"}];
  const original = formSnapshot(fields);
  fields[0].value = "Draft";
  assert.notEqual(formSnapshot(fields), original);
  fields[0].value = "Original";
  assert.equal(formSnapshot(fields), original);
});

test("bulk selections and hidden cursor values do not leave permanent dirty state", () => {
  const fields = [
    {name: "selected", type: "hidden", value: "a"},
    {name: "ids", type: "checkbox", value: "a", checked: false},
    {name: "operation", type: "select-one", value: "move"},
  ];
  const original = formSnapshot(fields);
  fields[0].value = "b";
  fields[1].checked = true;
  assert.equal(formSnapshot(fields), original);
  fields[1].checked = false;
  assert.equal(formSnapshot(fields), original);
});

test("mutation checkbox changes are real form changes", () => {
  const fields = [{name: "no_move", type: "checkbox", checked: false}];
  const original = formSnapshot(fields);
  fields[0].checked = true;
  assert.notEqual(formSnapshot(fields), original);
});

test("a failed move marks only its visible native form dirty", () => {
  const failure = {action: "move", id: "item-b"};
  assert.equal(failedFormMatches({action: "/actions/move", id: "item-b", hidden: false}, failure), true);
  assert.equal(failedFormMatches({action: "/actions/move", id: "item-b", hidden: true}, failure), false);
  assert.equal(failedFormMatches({action: "/actions/move", id: "item-a", hidden: false}, failure), false);
});

test("a failed board rename does not mark other board forms dirty", () => {
  const failure = {action: "board-rename", key: "ENG"};
  assert.equal(failedFormMatches({action: "/actions/board-rename", key: "ENG", hidden: false}, failure), true);
  assert.equal(failedFormMatches({action: "/actions/board-rename", key: "OPS", hidden: false}, failure), false);
});

test("canceling a submit then reverting the other form still protects the remaining draft", () => {
  const guard = createUnloadGuard(() => false);
  const dirty = new Set(["form-a", "form-b"]);
  assert.equal(guard.allowSubmit(dirty, "form-a"), false);
  dirty.delete("form-b");
  assert.equal(guard.shouldWarn(dirty.size > 0), true);
});

test("an accepted submit bypasses the draft warning for only one unload", () => {
  let confirmations = 0;
  const guard = createUnloadGuard(() => { confirmations++; return true; });
  assert.equal(guard.allowSubmit(new Set(["form-a", "form-b"]), "form-a"), true);
  assert.equal(confirmations, 1);
  assert.equal(guard.shouldWarn(true), false);
  assert.equal(guard.shouldWarn(true), true);
});

test("submitting the only draft needs no discard confirmation", () => {
  const guard = createUnloadGuard(() => { throw new Error("unexpected discard prompt"); });
  assert.equal(guard.allowSubmit(new Set(["form-a"]), "form-a"), true);
  assert.equal(guard.shouldWarn(true), false);
});
