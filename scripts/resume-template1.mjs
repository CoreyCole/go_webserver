#!/usr/bin/env node
// Minimal Template 1 generator adapted from Resumake v2.
// Vendored here so public/resume.pdf can be regenerated from public/resume.json.
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'

const inputPath = process.argv[2] || 'public/resume.json'
const outputPath = process.argv[3] || 'public/resume.pdf'

const resume = JSON.parse(fs.readFileSync(inputPath, 'utf8'))
resume.selectedTemplate = 1

const tex = template1(sanitizeResume(resume))
const tempDir = fs.mkdtempSync(path.join(os.tmpdir(), 'resume-template1-'))
const texPath = path.join(tempDir, 'resume.tex')
fs.writeFileSync(texPath, tex)

const tectonic = spawnSync('tectonic', ['--keep-logs', '--outdir', tempDir, texPath], {
  stdio: 'inherit'
})

if (tectonic.status !== 0) {
  const nix = spawnSync('nix', ['shell', 'nixpkgs#tectonic', '-c', 'tectonic', '--keep-logs', '--outdir', tempDir, texPath], {
    stdio: 'inherit'
  })
  if (nix.status !== 0) {
    process.exit(nix.status || 1)
  }
}

fs.mkdirSync(path.dirname(outputPath), { recursive: true })
fs.copyFileSync(path.join(tempDir, 'resume.pdf'), outputPath)
console.log(`generated ${outputPath}`)

function sanitizeResume(value) {
  if (Array.isArray(value)) {
    return value.map(sanitizeResume).filter((item) => item !== null && item !== '')
  }

  if (value && typeof value === 'object') {
    const output = {}
    for (const [key, child] of Object.entries(value)) {
      const sanitized = sanitizeResume(child)
      if (sanitized === null || sanitized === '') continue
      if (Array.isArray(sanitized) && sanitized.length === 0) continue
      output[key] = sanitized
    }
    return output
  }

  if (typeof value === 'string') {
    return escapeLatex(
      value
        .trim()
        .replace(/\s\s+/g, ' ')
        .replace(/→/g, '->')
        .replace(/[–—]/g, '-')
        .replace(/[“”]/g, '"')
        .replace(/[‘’]/g, "'")
    )
  }

  return value
}

function escapeLatex(value) {
  return value
    .replace(/\\/g, '\\textbackslash{}')
    .replace(/([{}&%$#_])/g, '\\$1')
    .replace(/~/g, '\\textasciitilde{}')
    .replace(/\^/g, '\\textasciicircum{}')
}

function stripIndent(strings, ...values) {
  const raw = strings.reduce((acc, string, index) => acc + string + (values[index] ?? ''), '')
  const lines = raw.replace(/^\n/, '').replace(/\n\s*$/, '').split('\n')
  const indents = lines.filter((line) => line.trim()).map((line) => line.match(/^\s*/)[0].length)
  const minIndent = Math.min(...indents, 0)
  return lines.map((line) => line.slice(minIndent)).join('\n')
}

function profileSection(basics) {
  if (!basics) return ''

  const { name, email, phone, location, website } = basics
  const address = location?.address || ''
  const websiteLine = website ? `\\href{${website}}{${website}}` : ''

  let line1 = name ? `{\\Huge \\scshape {${name}}}` : ''
  let line2 = [address, email, phone, websiteLine].filter(Boolean).join(' $\\cdot$ ')

  if (line1 && line2) {
    line1 += '\\\\'
    line2 += '\\\\'
  }

  return stripIndent`
    %==== Profile ====%
    \\vspace*{-10pt}
    \\begin{center}
      ${line1}
      ${line2}
    \\end{center}
  `
}

function awardsSection(awards, heading) {
  if (!awards) return ''

  return stripIndent`
    \\header{${heading || 'Awards'}}
    ${awards.map((award) => {
      const { title, summary, date, awarder } = award
      let line1 = ''
      let line2 = summary || ''

      if (title) line1 += `\\textbf{${title}}`
      if (awarder) line1 += ` \\hfill ${awarder}`
      if (date) line2 += ` \\hfill ${date}`
      if (line1) line1 += '\\\\'
      if (line2) line2 += '\\\\'

      return stripIndent`
        ${line1}
        ${line2}
        \\vspace*{2mm}
      `
    }).join('\n')}
  `
}

function workSection(work, heading) {
  if (!work) return ''

  return stripIndent`
    %==== Experience ====%
    \\header{${heading || 'Experience'}}
    \\vspace{1mm}

    ${work.map((job) => {
      const name = job.name || job.company
      const { position, location, startDate, endDate, highlights } = job
      let line1 = ''
      let line2 = ''
      let highlightLines = ''

      if (name) line1 += `\\textbf{${name}}`
      if (location) line1 += ` \\hfill ${location}`
      if (position) line2 += `\\textit{${position}}`
      if (startDate && endDate) line2 += ` \\hfill ${startDate} - ${endDate}`
      else if (startDate) line2 += ` \\hfill ${startDate} - Present`
      else if (endDate) line2 += ` \\hfill ${endDate}`
      if (line1) line1 += '\\\\'
      if (line2) line2 += '\\\\'

      if (highlights) {
        highlightLines = stripIndent`
          \\vspace{-1mm}
          \\begin{itemize} \\itemsep 1pt
            ${highlights.map((highlight) => `\\item ${highlight}`).join('\n')}
          \\end{itemize}
        `
      }

      return stripIndent`
        ${line1}
        ${line2}
        ${highlightLines}
      `
    }).join('\n')}
  `
}

function skillsSection(skills, heading) {
  if (!skills) return ''

  return stripIndent`
    \\header{${heading || 'Skills'}}
    \\begin{tabular}{ l l }
    ${skills.map((skill) => {
      const { name = 'Misc', keywords = [] } = skill
      return `${name}: & ${keywords.join(', ')} \\\\`
    }).join('\n')}
    \\end{tabular}
    \\vspace{2mm}
  `
}

function projectsSection(projects, heading) {
  if (!projects) return ''

  return stripIndent`
    \\header{${heading || 'Projects'}}
    ${projects.map((project) => {
      if (Object.keys(project).length === 0) return ''

      const { name, description, keywords, url } = project
      let line1 = ''
      let line2 = description || ''

      if (name) line1 += `{\\textbf{${name}}}`
      if (keywords) line1 += ` {\\sl ${keywords.join(', ')}} `
      if (url) line1 += `\\hfill \\href{${url}}{${url}}`
      if (line1) line1 += '\\\\'
      if (line2) line2 += '\\\\'

      return stripIndent`
        ${line1}
        ${line2}
        \\vspace*{2mm}
      `
    }).join('\n')}
  `
}

function educationSection(education, heading) {
  if (!education) return ''

  return stripIndent`
    %==== Education ====%
    \\header{${heading || 'Education'}}
    ${education.map((school) => {
      const { institution, location, studyType, area, score, startDate, endDate } = school
      let line1 = ''
      let line2 = ''

      if (institution) line1 += `\\textbf{${institution}}`
      if (location) line1 += `\\hfill ${location}`
      if (studyType) line2 += studyType
      if (area) line2 += studyType ? ` ${area}` : `Degree in ${area}`
      if (score) line2 += ` \\textit{GPA: ${score}}`
      if (startDate || endDate) {
        const gradLine = `${startDate || ''} - ${endDate || ''}`
        line2 += line2 ? ` \\hfill ${gradLine}` : gradLine
      }
      if (line1) line1 += '\\\\'
      if (line2) line2 += '\\\\'

      return stripIndent`
        ${line1}
        ${line2.trim()}
        \\vspace{2mm}
      `
    }).join('\n')}
  `
}

function resumeHeader() {
  return stripIndent`
    %\\renewcommand{\\encodingdefault}{cg}
    %\\renewcommand{\\rmdefault}{lgrcmr}

    \\def\\bull{\\vrule height 0.8ex width .7ex depth -.1ex }

    % DEFINITIONS FOR RESUME %%%%%%%%%%%%%%%%%%%%%%%

    \\newcommand{\\area} [2] {
        \\vspace*{-9pt}
        \\begin{verse}
            \\textbf{#1}   #2
        \\end{verse}
    }

    \\newcommand{\\lineunder} {
        \\vspace*{-8pt} \\\\
        \\hspace*{-18pt} \\hrulefill \\\\
    }

    \\newcommand{\\header} [1] {
        {\\hspace*{-18pt}\\vspace*{6pt} \\textsc{#1}}
        \\vspace*{-6pt} \\lineunder
    }

    \\newcommand{\\employer} [3] {
        { \\textbf{#1} (#2)\\\\ \\underline{\\textbf{\\emph{#3}}}\\\\  }
    }

    \\newcommand{\\contact} [3] {
        \\vspace*{-10pt}
        \\begin{center}
            {\\Huge \\scshape {#1}}\\\\
            #2 \\\\ #3
        \\end{center}
        \\vspace*{-8pt}
    }

    \\newenvironment{achievements}{
        \\begin{list}
            {$\\bullet$}{\\topsep 0pt \\itemsep -2pt}}{\\vspace*{4pt}
        \\end{list}
    }

    \\newcommand{\\schoolwithcourses} [4] {
        \\textbf{#1} #2 $\\bullet$ #3\\\\
        #4 \\\\
        \\vspace*{5pt}
    }

    \\newcommand{\\school} [4] {
        \\textbf{#1} #2 $\\bullet$ #3\\\\
        #4 \\\\
    }
    % END RESUME DEFINITIONS %%%%%%%%%%%%%%%%%%%%%%%
  `
}

function template1(values) {
  const headings = values.headings || {}
  const sections = values.sections || ['profile', 'awards', 'work', 'projects', 'skills', 'education']

  return stripIndent`
    \\documentclass[a4paper]{article}
    \\usepackage{fullpage}
    \\usepackage{amsmath}
    \\usepackage{amssymb}
    \\usepackage{textcomp}
    \\usepackage[utf8]{inputenc}
    \\usepackage[T1]{fontenc}
    \\textheight=10in
    \\pagestyle{empty}
    \\raggedright
    \\usepackage[left=0.8in,right=0.8in,bottom=0.8in,top=0.8in]{geometry}
    \\usepackage[hidelinks]{hyperref}

    ${resumeHeader()}

    \\begin{document}
    \\vspace*{-40pt}

    ${sections.map((section) => {
      switch (section) {
        case 'profile':
          return profileSection(values.basics)
        case 'education':
          return educationSection(values.education, headings.education)
        case 'work':
          return workSection(values.work, headings.work)
        case 'skills':
          return skillsSection(values.skills, headings.skills)
        case 'projects':
          return projectsSection(values.projects, headings.projects)
        case 'awards':
          return awardsSection(values.awards, headings.awards)
        default:
          return ''
      }
    }).join('\n\n')}

    \\ 
    \\end{document}
  `
}
