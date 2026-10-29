package main

import (
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"

	"github.com/overcat/osprey/internal/changes"
)

// palette are the colors of the kinds of change: the system's green, red,
// orange and blue, as macOS has them in light and dark, and the green and
// red legible as text on the background.
type palette struct {
	added, deleted, modified, renamed ui.Color
	addedText, deletedText            ui.Color
}

func colors(t *ui.Theme) palette {
	if t.Dark {
		return palette{ui.Hex("#30D158"), ui.Hex("#FF453A"), ui.Hex("#FF9F0A"), ui.Hex("#0A84FF"),
			ui.Hex("#30D158"), ui.Hex("#FF6961")}
	}
	return palette{ui.Hex("#34C759"), ui.Hex("#FF3B30"), ui.Hex("#FF9500"), ui.Hex("#007AFF"),
		ui.Hex("#248A3D"), ui.Hex("#D70015")}
}

func (p palette) of(s changes.Status) ui.Color {
	switch s {
	case changes.StatusAdded:
		return p.added
	case changes.StatusDeleted:
		return p.deleted
	case changes.StatusRenamed:
		return p.renamed
	}
	return p.modified
}

// nativeTheme is the system's theme as macOS's own apps look: the accent
// color the system's settings choose, light or dark as the system is,
// with macOS's grays for headers, fields and separators.
func nativeTheme(base *ui.Theme) *ui.Theme {
	t := *base
	if t.Dark {
		t.Background = ui.Hex("#1E1E1E")
		t.Surface = ui.Hex("#2A2A2C")
		t.SurfaceHover = ui.RGBA(255, 255, 255, 0.07)
		t.SurfacePressed = ui.RGBA(255, 255, 255, 0.12)
		t.Border = ui.Hex("#38383A")
		t.Text, t.TextMuted = ui.Hex("#F5F5F7"), ui.Hex("#98989D")
	} else {
		t.Background = ui.Hex("#FFFFFF")
		t.Surface = ui.Hex("#F6F6F7")
		t.SurfaceHover = ui.RGBA(0, 0, 0, 0.05)
		t.SurfacePressed = ui.RGBA(0, 0, 0, 0.08)
		t.Border = ui.Hex("#E3E3E6")
		t.Text, t.TextMuted = ui.Hex("#1D1D1F"), ui.Hex("#6E6E73")
	}
	p := colors(&t)
	t.Success, t.Danger, t.Warning = p.added, p.deleted, p.modified
	t.Radius = 6
	return &t
}

// sidebarFill is the sidebar's background where the window shows no
// material, as on Linux.
func sidebarFill(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.Hex("#252527")
	}
	return ui.Hex("#F0F0F2")
}

// fieldFill is the background of fields and segmented controls, which the
// sidebar's material shows through.
func fieldFill(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(255, 255, 255, 0.08)
	}
	return ui.RGBA(0, 0, 0, 0.06)
}

// codeTheme colors code as Xcode's default themes do, its changes in the
// system's green and red.
func codeTheme(t *ui.Theme) *editor.Theme {
	ct := editor.DefaultTheme(t)
	p := colors(t)
	ct.Added, ct.Deleted, ct.Modified = p.added, p.deleted, p.modified
	style := func(k editor.TokenKind, hex string, bold bool) {
		ct.Tokens[k] = editor.TokenStyle{Color: ui.Hex(hex), Bold: bold}
	}
	plain := func(k editor.TokenKind) { ct.Tokens[k] = editor.TokenStyle{Color: t.Text} }
	if t.Dark {
		style(editor.TokenKeyword, "#FC5FA3", true)
		style(editor.TokenString, "#FC6A5D", false)
		style(editor.TokenEscape, "#FC6A5D", false)
		style(editor.TokenNumber, "#D0BF69", false)
		style(editor.TokenConstant, "#D0BF69", false)
		style(editor.TokenType, "#D0A8FF", false)
		style(editor.TokenFunction, "#67B7A4", false)
		style(editor.TokenProperty, "#67B7A4", false)
		style(editor.TokenMeta, "#FD8F3F", false)
		style(editor.TokenComment, "#6C7986", false)
	} else {
		style(editor.TokenKeyword, "#9B2393", true)
		style(editor.TokenString, "#C41A16", false)
		style(editor.TokenEscape, "#C41A16", false)
		style(editor.TokenNumber, "#1C00CF", false)
		style(editor.TokenConstant, "#1C00CF", false)
		style(editor.TokenType, "#3900A0", false)
		style(editor.TokenFunction, "#326D74", false)
		style(editor.TokenProperty, "#326D74", false)
		style(editor.TokenMeta, "#643820", false)
		style(editor.TokenComment, "#5D6C79", false)
	}
	plain(editor.TokenOperator)
	return ct
}

var brandImage = func() *ui.Bitmap {
	b, err := ui.DecodeBitmap(iconPNG)
	if err != nil {
		panic(err)
	}
	return b
}()
