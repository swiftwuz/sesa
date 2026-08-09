import { execFile } from "node:child_process";
import { promisify } from "node:util";

import {
  parseContextList,
  parseCurrentRepository,
  type CurrentRepository,
} from "./protocol.js";

const execFileAsync = promisify(execFile);

export class SesaClient {
  async current(workingDirectory: string): Promise<CurrentRepository> {
    const output = await this.run(["current", "--json"], workingDirectory);
    return parseCurrentRepository(output);
  }

  async contexts(workingDirectory: string): Promise<string[]> {
    const output = await this.run(["list", "--json"], workingDirectory);
    return parseContextList(output).contexts;
  }

  async link(context: string, workingDirectory: string): Promise<void> {
    await this.run(["link", context], workingDirectory);
  }

  async openCode(context: string, repository: string): Promise<void> {
    await this.run(["code", context, repository], repository);
  }

  private async run(args: string[], workingDirectory: string): Promise<string> {
    const { stdout } = await execFileAsync("sesa", args, {
      cwd: workingDirectory,
      encoding: "utf8",
    });
    return stdout;
  }
}
