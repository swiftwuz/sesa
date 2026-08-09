import assert from "node:assert/strict";
import test from "node:test";

import {
  parseActiveContext,
  parseContextList,
  parseCurrentRepository,
} from "./protocol.js";

test("parses the version 1 mapped response", () => {
  const response = parseCurrentRepository(JSON.stringify({
    protocolVersion: 1,
    repository: "/repos/project",
    mapped: true,
    context: "personal"
  }));

  assert.equal(response.context, "personal");
});

test("rejects unsupported protocol versions", () => {
  assert.throws(() => parseCurrentRepository(JSON.stringify({
    protocolVersion: 2,
    repository: "/repos/project",
    mapped: false,
    context: null
  })));
});

test("only accepts safe active context names", () => {
  assert.equal(parseActiveContext("work"), "work");
  assert.equal(parseActiveContext("../work"), undefined);
  assert.equal(parseActiveContext(undefined), undefined);
});

test("parses a versioned context list", () => {
  const response = parseContextList(JSON.stringify({
    protocolVersion: 1,
    contexts: ["personal", "work"],
  }));

  assert.deepEqual(response.contexts, ["personal", "work"]);
});
