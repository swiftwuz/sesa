import assert from "node:assert/strict";
import test from "node:test";

import type { CurrentRepository } from "./protocol.js";
import { planSwitch } from "./switch.js";

const mappedWork: CurrentRepository = {
  protocolVersion: 2,
  repository: "/repos/project",
  mapped: true,
  contexts: ["work"],
};

test("opens a new window without relinking when the mapping already matches", () => {
  assert.deepEqual(planSwitch("personal", "work", mappedWork), {
    linkRepository: false,
    launchWindow: true,
  });
});

test("accepts either context when both are already allowed", () => {
  const bothAllowed: CurrentRepository = {
    protocolVersion: 2,
    repository: "/repos/project",
    mapped: true,
    contexts: ["personal", "work"],
  };
  assert.deepEqual(planSwitch("personal", "work", bothAllowed), {
    linkRepository: false,
    launchWindow: true,
  });
});

test("adds another allowed context without replacing the existing one", () => {
  assert.deepEqual(planSwitch("personal", "personal", mappedWork), {
    linkRepository: true,
    launchWindow: false,
  });
});

test("automatically links an unmapped repository", () => {
  const unmapped: CurrentRepository = {
    protocolVersion: 2,
    repository: "/repos/project",
    mapped: false,
    contexts: [],
  };
  assert.deepEqual(planSwitch("personal", "work", unmapped), {
    linkRepository: true,
    launchWindow: true,
  });
});
