import * as esbuild from "esbuild";

await esbuild.build({
  entryPoints: ["src/extension.ts"],
  bundle: true,
  external: ["vscode"],
  format: "cjs",
  platform: "node",
  outfile: "dist/extension.js",
  minify: true,
  sourcemap: true,
  sourcesContent: false,
  logLevel: "info"
});
