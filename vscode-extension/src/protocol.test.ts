import assert from "node:assert/strict";
import test from "node:test";

import {
  parseActiveContext,
  parseContextList,
  parseCurrentRepository,
} from "./protocol.js";

test("parses the version 2 mapped response", () => {
  const response = parseCurrentRepository(JSON.stringify({
    protocolVersion: 2,
    repository: "/repos/project",
    mapped: true,
    contexts: ["personal", "work"]
  }));

  assert.deepEqual(response.contexts, ["personal", "work"]);
});

test("rejects unsupported protocol versions", () => {
  assert.throws(() => parseCurrentRepository(JSON.stringify({
    protocolVersion: 3,
    repository: "/repos/project",
    mapped: false,
    contexts: []
  })));
});

test("rejects duplicate allowed contexts", () => {
  assert.throws(() => parseCurrentRepository(JSON.stringify({
    protocolVersion: 2,
    repository: "/repos/project",
    mapped: true,
    contexts: ["work", "work"],
  })));
});

test("only accepts safe active context names", () => {
  assert.equal(parseActiveContext("work"), "work");
  assert.equal(parseActiveContext("../work"), undefined);
  assert.equal(parseActiveContext(undefined), undefined);
});

test("parses a versioned context list", () => {
  const response = parseContextList(JSON.stringify({
    protocolVersion: 2,
    contexts: ["personal", "work"],
  }));

  assert.deepEqual(response.contexts, ["personal", "work"]);
});
