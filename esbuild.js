const esbuild = require("esbuild");

esbuild
  .build({
    entryPoints: ["react/main.tsx", "static/index.css", "static/build.css"],
    outdir: "react/dist",
    bundle: true,
    minify: true,
    plugins: [],
  })
  .then(() => console.log("⚡ Build complete! ⚡"))
  .catch(() => process.exit(1));
