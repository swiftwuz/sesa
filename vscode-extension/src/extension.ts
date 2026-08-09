import * as vscode from "vscode";

import { parseActiveContext } from "./protocol.js";
import { SesaClient } from "./sesa.js";
import { presentStatus, type StatusPresentation } from "./status.js";
import { planSwitch } from "./switch.js";

export function activate(extensionContext: vscode.ExtensionContext): void {
  const client = new SesaClient();
  const item = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Left,
    100,
  );
  item.name = "Sesa context";
  item.command = "sesa.switchContext";
  extensionContext.subscriptions.push(item);

  let refreshSequence = 0;
  const refresh = async (): Promise<void> => {
    const sequence = ++refreshSequence;
    const presentation = await resolvePresentation(client);
    if (sequence === refreshSequence) {
      render(item, presentation);
      item.show();
    }
  };

  extensionContext.subscriptions.push(
    vscode.commands.registerCommand("sesa.refreshStatus", refresh),
    vscode.commands.registerCommand("sesa.switchContext", async () => {
      await switchContext(client, refresh);
    }),
    vscode.workspace.onDidChangeWorkspaceFolders(refresh),
    vscode.window.onDidChangeWindowState((state) => {
      if (state.focused) {
        void refresh();
      }
    }),
  );

  void refresh();
}

async function resolvePresentation(client: SesaClient): Promise<StatusPresentation> {
  const activeContext = parseActiveContext(process.env.SESA_CONTEXT);
  const folders = vscode.workspace.workspaceFolders ?? [];
  if (activeContext === undefined) {
    return presentStatus({ workspaceCount: folders.length });
  }
  if (folders.length !== 1) {
    return presentStatus({ activeContext, workspaceCount: folders.length });
  }

  try {
    return presentStatus({
      activeContext,
      current: await client.current(folders[0]!.uri.fsPath),
      workspaceCount: folders.length,
    });
  } catch (error) {
    return presentStatus({
      activeContext,
      resolutionError: describeError(error),
      workspaceCount: folders.length,
    });
  }
}

async function switchContext(client: SesaClient, refresh: () => Promise<void>): Promise<void> {
  const repository = singleRepository();
  if (repository === undefined) {
    return;
  }

  try {
    const contexts = await client.contexts(repository);
    if (contexts.length === 0) {
      await vscode.window.showErrorMessage("No Sesa contexts found. Run sesa login <context> first.");
      return;
    }
    const selected = await pickContext(contexts, process.env.SESA_CONTEXT);
    if (selected === undefined) {
      return;
    }

    const current = await client.current(repository);
    const active = parseActiveContext(process.env.SESA_CONTEXT);
    const plan = planSwitch(active, selected, current);
    if (plan.confirmRemap) {
      if (!current.mapped || !(await confirmRemap(current.context, selected))) {
        return;
      }
    }
    if (plan.linkRepository) {
      await client.link(selected, repository);
    }
    if (plan.launchWindow) {
      await client.openCode(selected, repository);
      await vscode.window.showInformationMessage(`Opened a new isolated ${selected.toUpperCase()} window. Close this window when ready.`);
    } else {
      await refresh();
      await vscode.window.showInformationMessage(`This repository now uses ${selected.toUpperCase()}.`);
    }
  } catch (error) {
    await vscode.window.showErrorMessage(describeError(error));
  }
}

function singleRepository(): string | undefined {
  const folders = vscode.workspace.workspaceFolders ?? [];
  if (folders.length !== 1) {
    void vscode.window.showErrorMessage("Open exactly one repository before switching Sesa context.");
    return undefined;
  }
  return folders[0]!.uri.fsPath;
}

async function pickContext(contexts: string[], activeContext: string | undefined): Promise<string | undefined> {
  interface ContextItem extends vscode.QuickPickItem {
    context: string;
  }
  const items: ContextItem[] = contexts.map((context) =>
    context === activeContext
      ? { label: context.toUpperCase(), description: "Current window", context }
      : { label: context.toUpperCase(), context },
  );
  const selected = await vscode.window.showQuickPick(items, {
    title: "Open repository with Sesa context",
    placeHolder: "Choose an isolated Codex context",
  });
  return selected?.context;
}

async function confirmRemap(current: string, selected: string): Promise<boolean> {
  const choice = await vscode.window.showWarningMessage(
    `This repository is mapped to ${current.toUpperCase()}. Remap it to ${selected.toUpperCase()}?`,
    { modal: true },
    "Remap and continue",
  );
  return choice === "Remap and continue";
}

function render(
  item: vscode.StatusBarItem,
  presentation: StatusPresentation,
): void {
  item.text = presentation.text;
  item.tooltip = presentation.tooltip;
  item.backgroundColor = backgroundColor(presentation.emphasis);
}

function backgroundColor(
  emphasis: StatusPresentation["emphasis"],
): vscode.ThemeColor | undefined {
  if (emphasis === "error") {
    return new vscode.ThemeColor("statusBarItem.errorBackground");
  }
  if (emphasis === "warning") {
    return new vscode.ThemeColor("statusBarItem.warningBackground");
  }
  return undefined;
}

function describeError(error: unknown): string {
  if (isErrorWithCode(error) && error.code === "ENOENT") {
    return "Sesa CLI was not found in VS Code's PATH. Install sesa and reopen VS Code through sesa code.";
  }
  if (hasStderr(error) && error.stderr.trim() !== "") {
    return `Sesa command failed: ${error.stderr.trim()}`;
  }
  return error instanceof Error
    ? `Unable to resolve Sesa repository context: ${error.message}`
    : "Unable to resolve Sesa repository context.";
}

function isErrorWithCode(error: unknown): error is Error & { code: string } {
  return error instanceof Error && "code" in error && typeof error.code === "string";
}

function hasStderr(error: unknown): error is { stderr: string } {
  return typeof error === "object" && error !== null && "stderr" in error && typeof error.stderr === "string";
}

export function deactivate(): void {}
