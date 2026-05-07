package adt

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/aymanbagabas/go-udiff"
	"github.com/oisee/vibing-steampunk/pkg/adt/connection"
)

// --- Object Version History ---

// GetObjectVersions retrieves the version history (revisions) of an ABAP object.
// For classes, use opts.Include to get versions of a specific include.
//
// Supported object types: PROG, CLAS, INTF, FUNC, INCL, DDLS, BDEF, SRVD, TABL
func (c *Client) GetObjectVersions(ctx context.Context, objectType, name string, opts *GetSourceOptions) ([]Revision, error) {
	if err := c.checkSafety(OpRead, "GetObjectVersions"); err != nil {
		return nil, err
	}

	objectType = strings.ToUpper(objectType)
	name = strings.ToUpper(name)
	if opts == nil {
		opts = &GetSourceOptions{}
	}

	revisionURL, err := resolveRevisionURL(objectType, name, opts)
	if err != nil {
		return nil, fmt.Errorf("resolving revision URL for %s %s: %w", objectType, name, err)
	}

	resp, err := c.sendRequest(ctx, revisionURL, &connection.Request{
		Method: http.MethodGet,
		Accept: "application/atom+xml;type=feed",
	})
	if err != nil {
		return nil, fmt.Errorf("getting versions for %s %s: %w", objectType, name, err)
	}

	return ParseRevisionFeed(resp.Body)
}

// GetObjectVersionSource retrieves the source code of a specific object version.
// The versionURI comes from GetObjectVersions output (the URI field of a Revision entry).
func (c *Client) GetObjectVersionSource(ctx context.Context, versionURI string) (string, error) {
	if err := c.checkSafety(OpRead, "GetObjectVersionSource"); err != nil {
		return "", err
	}

	if versionURI == "" {
		return "", fmt.Errorf("versionURI is required")
	}

	resp, err := c.sendRequest(ctx, versionURI, &connection.Request{
		Method: http.MethodGet,
		Accept: "text/plain",
	})
	if err != nil {
		return "", fmt.Errorf("getting version source: %w", err)
	}

	return string(resp.Body), nil
}

// VersionDiff represents a unified diff between two object versions.
type VersionDiff struct {
	BaseVersionURI   string `json:"base_version_uri"`   // The base version URI (--- side)
	TargetVersionURI string `json:"target_version_uri"` // The target version URI (+++ side)
	Identical        bool   `json:"identical"`
	AddedLines       int    `json:"addedLines"`   // Lines added going from base → target ('+' lines)
	RemovedLines     int    `json:"removedLines"` // Lines removed going from base → target ('-' lines)
	Diff             string `json:"diff"`
}

// CompareObjectVersions compares two versions of an ABAP object and returns a unified diff.
// Like `git diff <base> <target>`: baseURI is the --- side, targetURI is the +++ side.
func (c *Client) CompareObjectVersions(ctx context.Context, baseURI, targetURI string) (*VersionDiff, error) {
	if err := c.checkSafety(OpRead, "CompareObjectVersions"); err != nil {
		return nil, err
	}

	baseSource, err := c.GetObjectVersionSource(ctx, baseURI)
	if err != nil {
		return nil, fmt.Errorf("getting base version source: %w", err)
	}

	targetSource, err := c.GetObjectVersionSource(ctx, targetURI)
	if err != nil {
		return nil, fmt.Errorf("getting target version source: %w", err)
	}

	baseLabel := fmt.Sprintf("base/%s", extractVersionLabel(baseURI))
	targetLabel := fmt.Sprintf("target/%s", extractVersionLabel(targetURI))

	result := &VersionDiff{
		BaseVersionURI:   baseURI,
		TargetVersionURI: targetURI,
		Identical:        baseSource == targetSource,
	}

	if result.Identical {
		result.Diff = "Sources are identical"
		return result, nil
	}

	result.Diff = udiff.Unified(baseLabel, targetLabel, baseSource, targetSource)

	for _, line := range strings.Split(result.Diff, "\n") {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			result.AddedLines++
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			result.RemovedLines++
		}
	}

	return result, nil
}

// resolveRevisionURL builds the ADT revision feed URL for a given object type.
//
// Key discovery: classes use /includes/{type}/versions (not /source/main/versions).
// Programs and other source objects use /source/main/versions.
// Interfaces also use /includes/main/versions (same pattern as classes).
func resolveRevisionURL(objectType, name string, opts *GetSourceOptions) (string, error) {
	encodedName := url.PathEscape(name)
	// DDIC objects (DDLS, TABL, SRVD) require lowercase names in the URL path
	encodedNameLower := url.PathEscape(strings.ToLower(name))

	switch objectType {
	case "PROG":
		return fmt.Sprintf("/sap/bc/adt/programs/programs/%s/source/main/versions", encodedName), nil
	case "CLAS":
		include := opts.Include
		if include == "" {
			include = "main"
		}
		// Classes always use /includes/{type}/versions — even for main
		return fmt.Sprintf("/sap/bc/adt/oo/classes/%s/includes/%s/versions", encodedName, include), nil
	case "INTF":
		// Interfaces use the same /includes/main/versions pattern as classes
		return fmt.Sprintf("/sap/bc/adt/oo/interfaces/%s/includes/main/versions", encodedName), nil
	case "FUNC":
		if opts.Parent == "" {
			return "", fmt.Errorf("parent (function group name) is required for FUNC type")
		}
		encodedParent := url.PathEscape(strings.ToUpper(opts.Parent))
		return fmt.Sprintf("/sap/bc/adt/functions/groups/%s/fmodules/%s/source/main/versions", encodedParent, encodedName), nil
	case "INCL":
		return fmt.Sprintf("/sap/bc/adt/programs/includes/%s/source/main/versions", encodedName), nil
	case "DDLS":
		// DDLS uses /versions directly (no /source/main prefix) and requires lowercase name
		return fmt.Sprintf("/sap/bc/adt/ddic/ddl/sources/%s/versions", encodedNameLower), nil
	case "BDEF":
		return fmt.Sprintf("/sap/bc/adt/bo/behaviordefinitions/%s/source/main/versions", encodedName), nil
	case "SRVD":
		return fmt.Sprintf("/sap/bc/adt/ddic/srvd/sources/%s/source/main/versions", encodedNameLower), nil
	case "TABL":
		return fmt.Sprintf("/sap/bc/adt/ddic/tables/%s/source/main/versions", encodedNameLower), nil
	default:
		return "", fmt.Errorf("unsupported object type for versions: %s (supported: PROG, CLAS, INTF, FUNC, INCL, DDLS, BDEF, SRVD, TABL)", objectType)
	}
}

// extractVersionLabel extracts a short label from a version URI for display.
// Handles two URI formats:
//   - PROG: .../source/main?version=5 → "v5"
//   - CLAS: .../versions/20161212091747/00000/content → "v00000"
func extractVersionLabel(uri string) string {
	if uri == "current" {
		return "current"
	}
	// Format 1: query param (programs)
	if idx := strings.Index(uri, "version="); idx >= 0 {
		rest := uri[idx+8:]
		if end := strings.IndexAny(rest, "&;"); end >= 0 {
			return "v" + rest[:end]
		}
		return "v" + rest
	}
	// Format 2: path-based (classes) — .../versions/{timestamp}/{version}/content
	if idx := strings.Index(uri, "/versions/"); idx >= 0 {
		rest := uri[idx+10:] // after "/versions/"
		parts := strings.Split(rest, "/")
		if len(parts) >= 2 {
			return "v" + parts[1] // version number (e.g., "00000")
		}
		if len(parts) == 1 {
			return "v" + parts[0]
		}
	}
	parts := strings.Split(uri, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return uri
}
