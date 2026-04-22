package adt

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"

	"github.com/oisee/vibing-steampunk/pkg/adt/connection"
)

// ADTEndpoint represents a single endpoint discovered from /sap/bc/adt/discovery.
type ADTEndpoint struct {
	// Path is the collection href (e.g., "/sap/bc/adt/programs/programs").
	Path string `json:"path"`
	// Accept lists the MIME types accepted by this endpoint.
	Accept []string `json:"accept,omitempty"`
}

// DiscoveredEndpoints maps endpoint paths to their ADTEndpoint metadata.
type DiscoveredEndpoints map[string]ADTEndpoint

// HasEndpoint checks whether the given path (or any discovered endpoint that
// is a prefix of it) exists in the discovery result.
func (d DiscoveredEndpoints) HasEndpoint(path string) bool {
	if len(d) == 0 {
		return false
	}
	// Exact match first
	if _, ok := d[path]; ok {
		return true
	}
	// Prefix match: check if any discovered endpoint is a prefix of the requested path
	for ep := range d {
		if len(ep) > 0 && len(path) >= len(ep) && path[:len(ep)] == ep {
			// Make sure we match at a path boundary
			if len(path) == len(ep) || path[len(ep)] == '/' {
				return true
			}
		}
	}
	return false
}

// DiscoverFeatures requests /sap/bc/adt/discovery and parses the AtomPub
// service document to build a map of available ADT endpoints.
func (c *Client) DiscoverFeatures(ctx context.Context) (DiscoveredEndpoints, error) {
	resp, err := c.sendRequest(ctx, "/sap/bc/adt/discovery", &connection.Request{
		Method: http.MethodGet,
		Accept: "application/atomsvc+xml",
	})
	if err != nil {
		return nil, fmt.Errorf("ADT discovery request failed: %w", err)
	}

	return parseDiscoveryResponse(resp.Body)
}

// --- AtomPub XML types for /sap/bc/adt/discovery ---

type atomService struct {
	XMLName    xml.Name        `xml:"service"`
	Workspaces []atomWorkspace `xml:"workspace"`
}

type atomWorkspace struct {
	Title       string           `xml:"title"`
	Collections []atomCollection `xml:"collection"`
}

type atomCollection struct {
	Href   string   `xml:"href,attr"`
	Title  string   `xml:"title"`
	Accept []string `xml:"accept"`
}

// parseDiscoveryResponse parses the AtomPub service document XML.
func parseDiscoveryResponse(data []byte) (DiscoveredEndpoints, error) {
	var svc atomService
	if err := xml.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("parsing ADT discovery XML: %w", err)
	}

	endpoints := make(DiscoveredEndpoints)
	for _, ws := range svc.Workspaces {
		for _, coll := range ws.Collections {
			if coll.Href == "" {
				continue
			}
			ep := ADTEndpoint{
				Path:   coll.Href,
				Accept: coll.Accept,
			}
			endpoints[coll.Href] = ep
		}
	}

	return endpoints, nil
}
