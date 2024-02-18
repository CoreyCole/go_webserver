package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Define structs to match JSON structure
type Resume struct {
	Awards    []Award     `json:"awards"`
	Work      []Work      `json:"work"`
	Skills    []Skill     `json:"skills"`
	Education []Education `json:"education"`
}

type Award struct {
	Title   string `json:"title"`
	Summary string `json:"summary"`
}

type Work struct {
	Company    string   `json:"position"` // positon and company
	Position   string   `json:"company"`  // swapped for pdf export
	Location   string   `json:"location"`
	StartDate  string   `json:"startDate"`
	EndDate    string   `json:"endDate"`
	Highlights []string `json:"highlights"`
}

type Skill struct {
	Name     string   `json:"name"`
	Keywords []string `json:"keywords"`
}

type Education struct {
	Institution string `json:"institution"`
	StudyType   string `json:"studyType"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	Location    string `json:"location"`
	Area        string `json:"area"`
}

// Main function to convert resume JSON to HTML
func ResumeJSONToHTML(path string) (string, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	var resume Resume
	if err := json.Unmarshal(bytes, &resume); err != nil {
		return "", err
	}

	// Initialize slice to hold HTML strings
	htmlSlice := []string{"<div>"}

	// Awards Section
	htmlSlice = append(htmlSlice, `<h3 class="myh myh-3">Resume</h3>`)
	for _, award := range resume.Awards {
		htmlSlice = append(
			htmlSlice,
			fmt.Sprintf(
				`<p class="font-bold text-lg">%s</p><p class="italic text-sm">%s</p>`,
				award.Title,
				award.Summary,
			),
		)
	}

	// Work Section
	htmlSlice = append(htmlSlice, `<h4 class="myh myh-4">Work Experience</h4>`)
	for _, work := range resume.Work {
		htmlSlice = append(
			htmlSlice,
			fmt.Sprintf(
				`<div class="flex justify-between items-center">
          <span class="text-start font-bold">%s</span>
          <span class="text-end italic">%s to %s</span>
        </div>
        <p>%s</p>`,
				work.Position,
				work.StartDate,
				work.EndDate,
				work.Company,
			),
		)
		for i, highlight := range work.Highlights {
			bottomMargin := ""
			if i == len(work.Highlights)-1 {
				bottomMargin = "mb-4"
			}
			htmlSlice = append(
				htmlSlice,
				fmt.Sprintf(`<li class="ml-4 text-lg %s">%s</li>`, bottomMargin, highlight),
			)
		}
	}

	// Skills Section
	htmlSlice = append(htmlSlice, `<h4 class="myh myh-4">Skills</h4>`)
	for _, skill := range resume.Skills {
		htmlSlice = append(
			htmlSlice,
			fmt.Sprintf(
				`<p><span class="font-bold">%s: </span>
        <span class="italic">%s</span><p>`,
				skill.Name,
				strings.Join(skill.Keywords, ", "),
			),
		)
	}

	// Education Section
	htmlSlice = append(htmlSlice, `<h4 class="myh myh-4">Education</h4>`)
	for _, edu := range resume.Education {
		htmlSlice = append(
			htmlSlice,
			fmt.Sprintf(
				`<div class="flex justify-between items-center">
          <span class="text-start font-bold">%s, %s</span>
          <span class="text-end italic">%s to %s</span>
        </div>
        <p>%s</p>`,
				edu.Institution,
				edu.StudyType,
				edu.StartDate,
				edu.EndDate,
				edu.Area,
			),
		)
	}

	htmlSlice = append(htmlSlice, "</div>") // Close the main div

	return strings.Join(htmlSlice, "\n"), nil
}
