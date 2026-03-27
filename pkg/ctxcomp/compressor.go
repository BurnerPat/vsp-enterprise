package ctxcomp

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
)

// Compressor orchestrates dependency extraction, source fetching, and contract generation.
type Compressor struct {
	provider SourceProvider
	maxDeps  int
}

// NewCompressor creates a new Compressor. maxDeps limits how many dependencies are resolved (default 20).
func NewCompressor(provider SourceProvider, maxDeps int) *Compressor {
	if maxDeps <= 0 {
		maxDeps = 20
	}
	return &Compressor{provider: provider, maxDeps: maxDeps}
}

// Compress extracts dependencies from source, fetches their contracts, and builds a prologue.
func (c *Compressor) Compress(ctx context.Context, source, objectName, objectType string) (*ContextResult, error) {
	deps := ExtractDependencies(source)

	// Filter self-references
	deps = filterSelf(deps, objectName)

	// Prioritize: custom (Z*/Y*) first, then SAP standard
	sort.Slice(deps, func(i, j int) bool {
		iCustom := isCustom(deps[i].Name)
		jCustom := isCustom(deps[j].Name)
		if iCustom != jCustom {
			return iCustom
		}
		return deps[i].Line < deps[j].Line
	})

	// Limit
	if len(deps) > c.maxDeps {
		deps = deps[:c.maxDeps]
	}

	contracts := c.fetchContracts(ctx, deps)

	prologue := formatPrologue(objectName, contracts)
	lines := strings.Count(prologue, "\n") + 1

	stats := ContextStats{
		DepsFound:  len(deps),
		TotalLines: lines,
	}
	for _, ct := range contracts {
		if ct.Error != "" {
			stats.DepsFailed++
		} else {
			stats.DepsResolved++
		}
	}

	return &ContextResult{
		SourceName:   objectName,
		Dependencies: deps,
		Contracts:    contracts,
		Prologue:     prologue,
		Stats:        stats,
	}, nil
}

func (c *Compressor) fetchContracts(ctx context.Context, deps []Dependency) []Contract {
	contracts := make([]Contract, len(deps))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5) // bounded parallelism

	for i, dep := range deps {
		wg.Add(1)
		go func(idx int, d Dependency) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fullSource, err := c.provider.GetSource(ctx, d.Kind, d.Name)
			if err != nil {
				contracts[idx] = Contract{
					Name:  d.Name,
					Kind:  d.Kind,
					Error: err.Error(),
				}
				return
			}

			compressed := ExtractContract(fullSource, d.Kind)
			contracts[idx] = Contract{
				Name:   d.Name,
				Kind:   d.Kind,
				Source: compressed,
			}
		}(i, dep)
	}

	wg.Wait()
	return contracts
}

func filterSelf(deps []Dependency, objectName string) []Dependency {
	upper := strings.ToUpper(objectName)
	filtered := make([]Dependency, 0, len(deps))
	for _, d := range deps {
		if d.Name != upper {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

func isCustom(name string) bool {
	return strings.HasPrefix(name, "Z") || strings.HasPrefix(name, "Y") ||
		strings.HasPrefix(name, "/Z") || strings.HasPrefix(name, "/Y")
}

func formatPrologue(objectName string, contracts []Contract) string {
	if len(contracts) == 0 {
		return ""
	}

	var resolved []Contract
	for _, c := range contracts {
		if c.Error == "" && c.Source != "" {
			resolved = append(resolved, c)
		}
	}

	if len(resolved) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("* === Dependency context for %s (%d deps) ===\n", objectName, len(resolved)))

	for _, c := range resolved {
		kindLabel := "class"
		switch c.Kind {
		case KindInterface:
			kindLabel = "interface"
		case KindFunction:
			kindLabel = "function module"
		}

		// Count methods in contract for info
		methodCount := strings.Count(strings.ToUpper(c.Source), "METHODS ")
		info := kindLabel
		if methodCount > 0 {
			info = fmt.Sprintf("%s, %d methods", kindLabel, methodCount)
		}

		sb.WriteString(fmt.Sprintf("\n* --- %s (%s) ---\n", c.Name, info))
		sb.WriteString(c.Source)
		sb.WriteString("\n")
	}

	return sb.String()
}
