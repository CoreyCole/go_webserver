/* eslint @typescript-eslint/no-var-requires: "off" */
const esbuild = require("esbuild")
const { performance } = require("perf_hooks")

const outdir = "public/react"

// use react from the CDN instead of bundling it with each individual component
// **DANGER** make sure the react version in package.json matches the version in the CDN
async function build() {
  try {
    const startTime = performance.now()
    await esbuild.build({
      entryPoints: ["react/index.ts"],
      globalName: "bundle",
      bundle: true,
      minify: false,
      outdir,
    })
    const duration = (performance.now() - startTime).toFixed(2)
    console.log(`⚡ TS->JS build complete in ${duration}ms! ⚡`)
  } catch (err) {
    console.error(err)
    process.exit(1)
  }
}
build()
