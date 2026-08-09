import assert from "node:assert/strict";
import test from "node:test";

import { presentStatus } from "./status.js";

test("shows a matching managed context", () => {
  const status = presentStatus({
    activeContext: "work",
    current: { protocolVersion: 2, repository: "/repos/project", mapped: true, contexts: ["personal", "work"] },
    workspaceCount: 1
  });

  assert.equal(status.text, "$(shield) SESA: WORK");
  assert.equal(status.emphasis, undefined);
});

test("makes a context mismatch unmistakable", () => {
  const status = presentStatus({
    activeContext: "personal",
    current: { protocolVersion: 2, repository: "/repos/project", mapped: true, contexts: ["work"] },
    workspaceCount: 1
  });

  assert.equal(status.text, "$(error) SESA: PERSONAL · NOT ALLOWED");
  assert.equal(status.emphasis, "error");
});

test("warns for unmapped and unmanaged windows", () => {
  const unmapped = presentStatus({
    activeContext: "personal",
    current: { protocolVersion: 2, repository: "/repos/project", mapped: false, contexts: [] },
    workspaceCount: 1
  });
  const unmanaged = presentStatus({ workspaceCount: 1 });

  assert.equal(unmapped.emphasis, "warning");
  assert.equal(unmanaged.text, "$(shield) SESA: UNMANAGED");
});

test("warns when repository resolution fails", () => {
  const status = presentStatus({
    activeContext: "work",
    resolutionError: "sesa is unavailable",
    workspaceCount: 1
  });

  assert.equal(status.text, "$(warning) SESA: WORK · CHECK FAILED");
  assert.equal(status.tooltip, "sesa is unavailable");
  assert.equal(status.emphasis, "warning");
});

test("does not pretend one mapping represents a multi-root workspace", () => {
  const status = presentStatus({ activeContext: "work", workspaceCount: 2 });

  assert.equal(status.text, "$(warning) SESA: WORK · MULTI-ROOT");
  assert.equal(status.emphasis, "warning");
});
