package annotate

import (
	"strings"
)

func ParseOperationMeta(lines []string) OperationMeta {
	var m OperationMeta

	for _, line := range lines {
		line = strings.TrimSpace(strings.TrimLeft(line, "/"))
		if line == "" || !strings.HasPrefix(line, "@") {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}

		switch strings.ToLower(fields[0]) {
		case "@id":
			if len(fields) > 1 {
				m.OperationID = fields[1]
			}
		case "@summary":
			m.Summary = strings.TrimSpace(line[len(fields[0]):])
		case "@description":
			m.Description = strings.TrimSpace(line[len(fields[0]):])
		case "@deprecated":
			m.Deprecated = true
		case "@tags":
			rest := strings.TrimSpace(line[len(fields[0]):])
			m.Tags = append(m.Tags, splitCSV(rest)...)
		}
	}

	return m
}

func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
