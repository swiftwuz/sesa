import type { CurrentRepository } from "./protocol.js";

export interface SwitchPlan {
  linkRepository: boolean;
  launchWindow: boolean;
  confirmRemap: boolean;
}

export function planSwitch(
  activeContext: string | undefined,
  selectedContext: string,
  current: CurrentRepository,
): SwitchPlan {
  const linkRepository =
    !current.mapped || current.context !== selectedContext;
  return {
    linkRepository,
    launchWindow: activeContext !== selectedContext,
    confirmRemap:
      current.mapped && current.context !== selectedContext,
  };
}
