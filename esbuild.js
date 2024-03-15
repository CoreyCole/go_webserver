/*eslint-env node*/

const esbuild = require("esbuild")
const { performance } = require("perf_hooks")
const outdir = "public/react"

// use react from the CDN instead of bundling it with each individual component
// **DANGER** make sure the react version in package.json matches the version in the CDN
async function build() {
  const startTime = performance.now()
  try {
    await esbuild.build({
      entryPoints: ["react/main.tsx"],
      outdir: outdir,
      bundle: false,
      minify: false,
      plugins: [],
    })
    await esbuild.build({
      entryPoints: ["react/components/dropdown.tsx"],
      outdir: `${outdir}/components`,
      bundle: false,
      minify: false,
      plugins: [],
      jsxFactory: "React.createElement",
      jsxFragment: "React.Fragment",
    })
    const endTime = performance.now()
    const duration = (endTime - startTime).toFixed(2)
    console.log(`⚡ TS->JS build complete in ${duration}ms! ⚡`)
  } catch (err) {
    console.error(err)
    process.exit(1)
  }
}
build()
