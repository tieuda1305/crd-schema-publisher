package index

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type schemaEntry struct {
	Name string
	Path string
}

type groupData struct {
	Name    string
	Schemas []schemaEntry
}

type indexData struct {
	Groups     []groupData
	TotalCount int
	UpdatedAt  string
}

const indexTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Kubernetes CRD Schemas</title>
<style>
  :root { --bg: #fff; --fg: #1a1a1a; --accent: #0066cc; --border: #e0e0e0; --details-bg: #f8f8f8; }
  @media (prefers-color-scheme: dark) {
    :root { --bg: #1a1a1a; --fg: #e0e0e0; --accent: #66b3ff; --border: #333; --details-bg: #222; }
  }
  * { box-sizing: border-box; margin: 0; padding: 0; }
  body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: var(--bg); color: var(--fg); max-width: 900px; margin: 0 auto; padding: 2rem 1rem; }
  h1 { margin-bottom: 0.5rem; }
  .meta { color: #888; margin-bottom: 2rem; font-size: 0.9rem; }
  details { border: 1px solid var(--border); border-radius: 4px; margin-bottom: 0.5rem; }
  summary { padding: 0.75rem 1rem; cursor: pointer; font-weight: 600; background: var(--details-bg); border-radius: 4px; }
  summary:hover { opacity: 0.8; }
  .schemas { padding: 0.5rem 1rem 0.75rem; }
  .schemas a { display: block; padding: 0.25rem 0; color: var(--accent); text-decoration: none; font-size: 0.9rem; }
  .schemas a:hover { text-decoration: underline; }
  .count { font-weight: normal; color: #888; font-size: 0.85rem; }
</style>
</head>
<body>
<h1>Kubernetes CRD Schemas</h1>
<p class="meta">{{.TotalCount}} CRD schemas &middot; Updated {{.UpdatedAt}}</p>
{{range .Groups}}
<details>
<summary>{{.Name}} <span class="count">({{len .Schemas}})</span></summary>
<div class="schemas">
{{range .Schemas}}<a href="/{{.Path}}">{{.Name}}</a>
{{end}}</div>
</details>
{{end}}
</body>
</html>`

func Generate(outputDir string) error {
	groups := map[string][]schemaEntry{}

	entries, err := os.ReadDir(outputDir)
	if err != nil {
		return fmt.Errorf("reading output dir: %w", err)
	}

	totalCount := 0
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name() == "master-standalone" {
			continue
		}
		groupName := entry.Name()
		groupDir := filepath.Join(outputDir, groupName)
		files, err := os.ReadDir(groupDir)
		if err != nil {
			return fmt.Errorf("reading group dir %s: %w", groupName, err)
		}
		for _, f := range files {
			if f.IsDir() || !strings.HasSuffix(f.Name(), ".json") {
				continue
			}
			groups[groupName] = append(groups[groupName], schemaEntry{
				Name: f.Name(),
				Path: groupName + "/" + f.Name(),
			})
			totalCount++
		}
	}

	var sortedGroups []groupData
	for name, schemas := range groups {
		sort.Slice(schemas, func(i, j int) bool { return schemas[i].Name < schemas[j].Name })
		sortedGroups = append(sortedGroups, groupData{Name: name, Schemas: schemas})
	}
	sort.Slice(sortedGroups, func(i, j int) bool { return sortedGroups[i].Name < sortedGroups[j].Name })

	tmpl, err := template.New("index").Parse(indexTemplate)
	if err != nil {
		return fmt.Errorf("parsing template: %w", err)
	}

	f, err := os.Create(filepath.Join(outputDir, "index.html"))
	if err != nil {
		return fmt.Errorf("creating index.html: %w", err)
	}
	defer f.Close()

	return tmpl.Execute(f, indexData{
		Groups:     sortedGroups,
		TotalCount: totalCount,
		UpdatedAt:  time.Now().UTC().Format("2006-01-02 15:04 UTC"),
	})
}
