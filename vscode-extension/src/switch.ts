import type { CurrentRepository } from "./protocol.js";

export interface SwitchPlan {
  linkRepository: boolean;
  launchWindow: boolean;
}

export function planSwitch(
  activeContext: string | undefined,
  selectedContext: string,
  current: CurrentRepository,
): SwitchPlan {
  const linkRepository = !current.contexts.includes(selectedContext);
  return {
    linkRepository,
    launchWindow: activeContext !== selectedContext,
  };
}
