import type { CurrentRepository } from "./protocol.js";

export type Emphasis = "warning" | "error";

export interface StatusPresentation {
  text: string;
  tooltip: string;
  emphasis?: Emphasis;
}

export interface StatusInput {
  activeContext?: string;
  current?: CurrentRepository;
  resolutionError?: string;
  workspaceCount: number;
}

export function presentStatus(input: StatusInput): StatusPresentation {
  if (input.activeContext === undefined) {
    return {
      text: "$(shield) SESA: UNMANAGED",
      tooltip: "This VS Code window was not launched by Sesa.",
    };
  }

  const active = input.activeContext.toUpperCase();
  if (input.workspaceCount > 1) {
    return {
      text: `$(warning) SESA: ${active} · MULTI-ROOT`,
      tooltip:
        "Sesa cannot safely represent multiple repository mappings in one status item.",
      emphasis: "warning",
    };
  }
  if (input.workspaceCount === 0) {
    return {
      text: `$(shield) SESA: ${active}`,
      tooltip: `Active Codex context: ${active}. No repository is open.`,
    };
  }
  if (input.resolutionError !== undefined) {
    return {
      text: `$(warning) SESA: ${active} · CHECK FAILED`,
      tooltip: input.resolutionError,
      emphasis: "warning",
    };
  }
  if (!input.current?.mapped) {
    return {
      text: `$(warning) SESA: ${active} · UNMAPPED`,
      tooltip:
        input.current === undefined
          ? "This repository is not mapped to a Sesa context."
          : `Repository: ${input.current.repository}\nNo Sesa context mapping.`,
      emphasis: "warning",
    };
  }

  const allowed = input.current.contexts.map((context) => context.toUpperCase());
  if (!input.current.contexts.includes(input.activeContext)) {
    return {
      text: `$(error) SESA: ${active} · NOT ALLOWED`,
      tooltip: `Repository: ${input.current.repository}\nActive context: ${active}\nAllowed contexts: ${allowed.join(", ")}.`,
      emphasis: "error",
    };
  }
  return {
    text: `$(shield) SESA: ${active}`,
    tooltip: `Repository: ${input.current.repository}\nActive context: ${active}\nAllowed contexts: ${allowed.join(", ")}.`,
  };
}
