import { execFile } from "node:child_process";
import { promisify } from "node:util";
import * as vscode from "vscode";

import { parseActiveContext, parseCurrentRepository } from "./protocol.js";
import { presentStatus, type StatusPresentation } from "./status.js";

const execFileAsync = promisify(execFile);

export function activate(extensionContext: vscode.ExtensionContext): void {
  const item = vscode.window.createStatusBarItem(
    vscode.StatusBarAlignment.Left,
    100,
  );
  item.name = "Sesa context";
  item.command = "sesa.refreshStatus";
  extensionContext.subscriptions.push(item);

  let refreshSequence = 0;
  const refresh = async (): Promise<void> => {
    const sequence = ++refreshSequence;
    const presentation = await resolvePresentation();
    if (sequence === refreshSequence) {
      render(item, presentation);
      item.show();
    }
  };

  extensionContext.subscriptions.push(
    vscode.commands.registerCommand("sesa.refreshStatus", refresh),
    vscode.workspace.onDidChangeWorkspaceFolders(refresh),
    vscode.window.onDidChangeWindowState((state) => {
      if (state.focused) {
        void refresh();
      }
    }),
  );

  void refresh();
}

async function resolvePresentation(): Promise<StatusPresentation> {
  const activeContext = parseActiveContext(process.env.SESA_CONTEXT);
  const folders = vscode.workspace.workspaceFolders ?? [];
  if (activeContext === undefined) {
    return presentStatus({ workspaceCount: folders.length });
  }
  if (folders.length !== 1) {
    return presentStatus({ activeContext, workspaceCount: folders.length });
  }

  try {
    const { stdout } = await execFileAsync("sesa", ["current", "--json"], {
      cwd: folders[0]!.uri.fsPath,
      encoding: "utf8",
    });
    return presentStatus({
      activeContext,
      current: parseCurrentRepository(stdout),
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
  return error instanceof Error
    ? `Unable to resolve Sesa repository context: ${error.message}`
    : "Unable to resolve Sesa repository context.";
}

export function deactivate(): void {}
