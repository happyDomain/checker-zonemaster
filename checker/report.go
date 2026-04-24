package checker

import (
	"encoding/json"
	"fmt"
	"html/template"
	"sort"
	"strings"

	sdk "git.happydns.org/checker-sdk-go/checker"
)

// ── HTML report ───────────────────────────────────────────────────────────────

// zmLevelDisplayOrder defines the severity order used for sorting and display.
var zmLevelDisplayOrder = []string{"CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG"}

var zmLevelRank = func() map[string]int {
	m := make(map[string]int, len(zmLevelDisplayOrder))
	for i, l := range zmLevelDisplayOrder {
		m[l] = len(zmLevelDisplayOrder) - i
	}
	return m
}()

type zmLevelCount struct {
	Level string
	Count int
}

type zmModuleGroup struct {
	Name     string
	Position int // first-seen index, used as tiebreaker in sort
	Results  []ZonemasterTestResult
	Levels   []zmLevelCount // sorted by severity desc, zeros omitted
	Worst    string
	Open     bool
}

type zmTemplateData struct {
	Domain    string
	CreatedAt string
	HashID    string
	Language  string
	Modules   []zmModuleGroup
	Totals    []zmLevelCount // sorted by severity desc, zeros omitted
}

var zonemasterHTMLTemplate = template.Must(
	template.New("zonemaster").
		Funcs(template.FuncMap{
			"badgeClass": func(level string) string {
				switch strings.ToUpper(level) {
				case "CRITICAL":
					return "badge-critical"
				case "ERROR":
					return "badge-error"
				case "WARNING":
					return "badge-warning"
				case "NOTICE":
					return "badge-notice"
				case "INFO":
					return "badge-info"
				default:
					return "badge-debug"
				}
			},
		}).
		Parse(`<!DOCTYPE html>
<html lang="{{.Language}}">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Zonemaster{{if .Domain}}, {{.Domain}}{{end}}</title>
<style>
*, *::before, *::after { box-sizing: border-box; }
:root {
  font-family: system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  font-size: 14px;
  line-height: 1.5;
  color: #1f2937;
  background: #f3f4f6;
}
body { margin: 0; padding: 1rem; }
a { color: inherit; }
code { font-family: ui-monospace, monospace; font-size: .9em; }

/* Header card */
.hd {
  background: #fff;
  border-radius: 10px;
  padding: 1rem 1.25rem 1.1rem;
  margin-bottom: .75rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.08);
}
.hd h1 { margin: 0 0 .2rem; font-size: 1.15rem; font-weight: 700; }
.hd .meta { color: #6b7280; font-size: .82rem; margin-bottom: .6rem; }
.totals { display: flex; gap: .35rem; flex-wrap: wrap; }

/* Badges */
.badge {
  display: inline-flex; align-items: center;
  padding: .18em .55em;
  border-radius: 9999px;
  font-size: .72rem; font-weight: 700;
  letter-spacing: .02em; white-space: nowrap;
}
.badge-critical { background: #fee2e2; color: #991b1b; }
.badge-error    { background: #ffedd5; color: #9a3412; }
.badge-warning  { background: #fef3c7; color: #92400e; }
.badge-notice   { background: #e0f2fe; color: #075985; }
.badge-info     { background: #dbeafe; color: #1e40af; }
.badge-debug    { background: #f3f4f6; color: #4b5563; }

/* Accordion */
details {
  background: #fff;
  border-radius: 8px;
  margin-bottom: .45rem;
  box-shadow: 0 1px 3px rgba(0,0,0,.07);
  overflow: hidden;
}
summary {
  display: flex; align-items: center; gap: .5rem;
  padding: .65rem 1rem;
  cursor: pointer;
  user-select: none;
  list-style: none;
}
summary::-webkit-details-marker { display: none; }
summary::before {
  content: "▶";
  font-size: .65rem;
  color: #9ca3af;
  transition: transform .15s;
  flex-shrink: 0;
}
details[open] > summary::before { transform: rotate(90deg); }
.mod-name { font-weight: 600; flex: 1; font-size: .9rem; }
.mod-badges { display: flex; gap: .25rem; flex-wrap: wrap; }

/* Result rows */
.results { border-top: 1px solid #f3f4f6; }
.row {
  display: grid;
  grid-template-columns: max-content 1fr;
  gap: .6rem;
  padding: .45rem 1rem;
  border-bottom: 1px solid #f9fafb;
  align-items: start;
}
.row:last-child { border-bottom: none; }
.row-msg { color: #374151; }
.row-tc  { font-size: .75rem; color: #9ca3af; }
</style>
</head>
<body>

<div class="hd">
  <h1>Zonemaster{{if .Domain}}, <code>{{.Domain}}</code>{{end}}</h1>
  <div class="meta">
    {{- if .CreatedAt}}Run at {{.CreatedAt}}{{end -}}
    {{- if and .CreatedAt .HashID}} &middot; {{end -}}
    {{- if .HashID}}ID: <code>{{.HashID}}</code>{{end -}}
  </div>
  <div class="totals">
    {{- range .Totals}}
    <span class="badge {{badgeClass .Level}}">{{.Level}}&nbsp;{{.Count}}</span>
    {{- end}}
  </div>
</div>

{{range .Modules -}}
<details{{if .Open}} open{{end}}>
  <summary>
    <span class="mod-name">{{.Name}}</span>
    <span class="mod-badges">
      {{- range .Levels}}
      <span class="badge {{badgeClass .Level}}">{{.Count}}</span>
      {{- end}}
    </span>
  </summary>
  <div class="results">
    {{- range .Results}}
    <div class="row">
      <span class="badge {{badgeClass .Level}}">{{.Level}}</span>
      <div>
        <div class="row-msg">{{.Message}}</div>
        {{- if .Testcase}}<div class="row-tc">{{.Testcase}}</div>{{end}}
      </div>
    </div>
    {{- end}}
  </div>
</details>
{{end -}}

</body>
</html>`),
)

// GetHTMLReport implements sdk.CheckerHTMLReporter.
func (p *zonemasterProvider) GetHTMLReport(ctx sdk.ReportContext) (string, error) {
	var data ZonemasterData
	if err := json.Unmarshal(ctx.Data(), &data); err != nil {
		return "", fmt.Errorf("failed to unmarshal zonemaster results: %w", err)
	}

	// Group results by module, preserving first-seen order.
	moduleOrder := []string{}
	moduleMap := map[string][]ZonemasterTestResult{}
	for _, r := range data.Results {
		if _, seen := moduleMap[r.Module]; !seen {
			moduleOrder = append(moduleOrder, r.Module)
		}
		moduleMap[r.Module] = append(moduleMap[r.Module], r)
	}

	totalCounts := map[string]int{}

	var modules []zmModuleGroup
	for _, name := range moduleOrder {
		rs := moduleMap[name]
		counts := map[string]int{}
		for _, r := range rs {
			lvl := strings.ToUpper(r.Level)
			counts[lvl]++
			totalCounts[lvl]++
		}

		// Find worst level and build sorted level-count slice.
		worst := ""
		worstRank := -1
		var levels []zmLevelCount
		for _, l := range zmLevelDisplayOrder {
			if n, ok := counts[l]; ok && n > 0 {
				levels = append(levels, zmLevelCount{Level: l, Count: n})
				if zmLevelRank[l] > worstRank {
					worstRank = zmLevelRank[l]
					worst = l
				}
			}
		}
		// Append any unknown levels last.
		for l, n := range counts {
			if _, known := zmLevelRank[l]; !known {
				levels = append(levels, zmLevelCount{Level: l, Count: n})
			}
		}

		modules = append(modules, zmModuleGroup{
			Name:     name,
			Position: len(modules),
			Results:  rs,
			Levels:   levels,
			Worst:    worst,
			Open:     worst == "CRITICAL" || worst == "ERROR",
		})
	}

	// Sort modules: most severe first, then by original appearance order.
	sort.Slice(modules, func(i, j int) bool {
		ri, rj := zmLevelRank[modules[i].Worst], zmLevelRank[modules[j].Worst]
		if ri != rj {
			return ri > rj
		}
		return modules[i].Position < modules[j].Position
	})

	// Build sorted totals slice.
	var totals []zmLevelCount
	for _, l := range zmLevelDisplayOrder {
		if n, ok := totalCounts[l]; ok && n > 0 {
			totals = append(totals, zmLevelCount{Level: l, Count: n})
		}
	}

	domain := ""
	if d, ok := data.Params["domain"]; ok {
		domain = fmt.Sprintf("%v", d)
	}

	lang := data.Language
	if lang == "" {
		lang = "en"
	}

	td := zmTemplateData{
		Domain:    domain,
		CreatedAt: data.CreatedAt,
		HashID:    data.HashID,
		Language:  lang,
		Modules:   modules,
		Totals:    totals,
	}

	var buf strings.Builder
	if err := zonemasterHTMLTemplate.Execute(&buf, td); err != nil {
		return "", fmt.Errorf("failed to render zonemaster HTML report: %w", err)
	}
	return buf.String(), nil
}
