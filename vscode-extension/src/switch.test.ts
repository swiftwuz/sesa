import assert from "node:assert/strict";
import test from "node:test";

import type { CurrentRepository } from "./protocol.js";
import { planSwitch } from "./switch.js";

const mappedWork: CurrentRepository = {
  protocolVersion: 1,
  repository: "/repos/project",
  mapped: true,
  context: "work",
};

test("opens a new window without relinking when the mapping already matches", () => {
  assert.deepEqual(planSwitch("personal", "work", mappedWork), {
    linkRepository: false,
    launchWindow: true,
    confirmRemap: false,
  });
});

test("requires confirmation before replacing a different mapping", () => {
  assert.deepEqual(planSwitch("personal", "personal", mappedWork), {
    linkRepository: true,
    launchWindow: false,
    confirmRemap: true,
  });
});

test("automatically links an unmapped repository", () => {
  const unmapped: CurrentRepository = {
    protocolVersion: 1,
    repository: "/repos/project",
    mapped: false,
    context: null,
  };
  assert.deepEqual(planSwitch("personal", "work", unmapped), {
    linkRepository: true,
    launchWindow: true,
    confirmRemap: false,
  });
});
