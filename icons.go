package main

import "github.com/egoist/mygo/ui"

// Icons of Lucide (lucide.dev, ISC license), drawn in the text's color.
func stroked(paths string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">` + paths + `</svg>`))
}

var (
	iconFocus      = stroked(`<path d="M8 3H3v5m13-5h5v5M3 16v5h5m13-5v5h-5"/>`)
	iconSplit      = stroked(`<rect x="3" y="4" width="18" height="16" rx="2"/><path d="M12 4v16"/>`)
	iconExpand     = stroked(`<path d="M4 9h16M4 15h16m5-10 3-3 3 3m-6 14 3 3 3-3"/>`)
	iconChevron    = stroked(`<path d="m9 18 6-6-6-6"/>`)
	iconChevronUp  = stroked(`<path d="m18 15-6-6-6 6"/>`)
	iconChevronDn  = stroked(`<path d="m6 9 6 6 6-6"/>`)
	iconFolderOpen = stroked(`<path d="m6 14 1.5-2.9A2 2 0 0 1 9.24 10H20a2 2 0 0 1 1.94 2.5l-1.54 6a2 2 0 0 1-1.95 1.5H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h3.9a2 2 0 0 1 1.69.9l.81 1.2a2 2 0 0 0 1.67.9H18a2 2 0 0 1 2 2v2"/>`)
	iconTree       = stroked(`<path d="M21 12h-8"/><path d="M21 6H8"/><path d="M21 18h-8"/><path d="M3 6v4c0 1.1.9 2 2 2h3"/><path d="M3 10v6c0 1.1.9 2 2 2h3"/>`)
	iconList       = stroked(`<path d="M3 12h.01"/><path d="M3 18h.01"/><path d="M3 6h.01"/><path d="M8 12h13"/><path d="M8 18h13"/><path d="M8 6h13"/>`)
	iconRefresh    = stroked(`<path d="M21 12a9 9 0 1 1-9-9c2.52 0 4.93 1 6.74 2.74L21 8"/><path d="M21 3v5h-5"/>`)
	iconWrap       = stroked(`<path d="m16 16-2 2 2 2"/><path d="M3 12h15a3 3 0 1 1 0 6h-4"/><path d="M3 18h7"/><path d="M3 6h18"/>`)
	iconCheck      = stroked(`<path d="M20 6 9 17l-5-5"/>`)
	iconCheckDone  = stroked(`<circle cx="12" cy="12" r="9"/><path d="m8.5 12 2.5 2.5 4.5-5"/>`)
	iconBranch     = stroked(`<path d="M6 3v12"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><path d="M18 9a9 9 0 0 1-9 9"/>`)
	iconFile       = stroked(`<path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z"/><path d="M14 2v4a2 2 0 0 0 2 2h4"/>`)
	iconWarning    = stroked(`<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3"/><path d="M12 9v4"/><path d="M12 17h.01"/>`)
	iconSearch     = stroked(`<circle cx="11" cy="11" r="7"/><path d="m20 20-4-4"/>`)
	iconClear      = stroked(`<path d="M18 6 6 18"/><path d="m6 6 12 12"/>`)
	iconSidebar    = stroked(`<rect width="18" height="18" x="3" y="3" rx="2"/><path d="M9 3v18"/>`)
)
