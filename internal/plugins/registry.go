// Package plugins is the extension point for Netbox plugin endpoints.
//
// Netbox plugins live under /api/plugins/<name>/ and are not part of the core
// OpenAPI surface. Two integration paths:
//
//  1. Named plugins — implement Plugin and call Register() in an init().
//     The plugin gets its own cobra subtree and (optionally) a bubbletea view.
//  2. Generic passthrough — for any plugin not wrapped yet, the user runs:
//     `nbcli plugin <name> get <subpath>` and we forward the call.
//
// This package only defines the interface and registry. Each concrete plugin
// lives in its own subpackage (e.g. internal/plugins/wireless/) and is wired
// in by importing it for side effects from cmd/nbcli/main.go.
//
// Concurrency note: Register() is meant for package init() (sequential by the
// Go runtime) and the registry is read-only after main() starts. No mutex.
package plugins

import (
	"sort"

	"github.com/spf13/cobra"
)

// Plugin is implemented by each named plugin integration.
type Plugin interface {
	// Name is the URL slug under /api/plugins/<name>/. Must be stable.
	Name() string

	// Title is a short human label for help text ("Wireless Controllers").
	Title() string

	// Commands returns subcommands mounted under `nbcli plugin <name>`.
	// Each command receives the shared netbox.Client via cobra's command
	// context — see cmd/plugin.go.
	Commands() []*cobra.Command
}

// Registry is a process-wide set of named plugins, populated via init().
type Registry struct {
	plugins map[string]Plugin
}

var global = &Registry{plugins: map[string]Plugin{}}

// Default returns the process-wide registry.
func Default() *Registry { return global }

// Register adds p to the registry. Panics on duplicate name — a plugin name
// collision is a programming error and we want it loud at startup.
// Intended to be called from a plugin package's init().
func Register(p Plugin) {
	if _, dup := global.plugins[p.Name()]; dup {
		panic("plugins: duplicate registration for " + p.Name())
	}
	global.plugins[p.Name()] = p
}

// List returns all registered plugins sorted by Name. The slice is a copy;
// callers may mutate it freely.
func (r *Registry) List() []Plugin {
	out := make([]Plugin, 0, len(r.plugins))
	for _, p := range r.plugins {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name() < out[j].Name() })
	return out
}

// Get returns a plugin by name, or nil if not registered.
func (r *Registry) Get(name string) Plugin {
	return r.plugins[name]
}
