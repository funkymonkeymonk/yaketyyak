package tui

import "github.com/charmbracelet/lipgloss"

const (
	FooterHeight             = 1
	HeaderHeight             = 1
	DefaultPreviewWidthRatio = 0.50
)

type ProgramContext struct {
	ScreenWidth  int
	ScreenHeight int

	MainContentWidth  int
	MainContentHeight int

	PreviewWidth  int
	PreviewHeight int
	SidebarOpen   bool

	Styles Styles
}

type Styles struct {
	Footer  FooterStyles
	Sidebar SectionStyles
	Common  CommonStyles
}

type FooterStyles struct {
	BarStyle    lipgloss.Style
	LeftStyle   lipgloss.Style
	RightStyle  lipgloss.Style
	KeyStyle    lipgloss.Style
	ActionStyle lipgloss.Style
}

type SectionStyles struct {
	ContainerStyle lipgloss.Style
	KeyStyle       lipgloss.Style
	SelectStyle    lipgloss.Style
}

type CommonStyles struct {
	MainTextStyle lipgloss.Style
	FooterStyle   lipgloss.Style
	HelpStyle     lipgloss.Style
	OverlayStyle  lipgloss.Style
}

func defaultContext() ProgramContext {
	return ProgramContext{
		SidebarOpen: true,
		Styles:      defaultStyles(),
	}
}

func (m *Model) syncMainContentDimensions() {
	ctx := &m.ctx

	ctx.ScreenWidth = m.width
	ctx.ScreenHeight = m.height

	ctx.MainContentHeight = m.height - FooterHeight - HeaderHeight
	if ctx.MainContentHeight < 1 {
		ctx.MainContentHeight = 1
	}

	if !ctx.SidebarOpen {
		ctx.MainContentWidth = m.width
		ctx.PreviewWidth = 0
		ctx.PreviewHeight = 0
	} else {
		previewW := int(float64(m.width) * DefaultPreviewWidthRatio)
		treeW := m.width - previewW - 1
		if treeW < 30 {
			treeW = 30
			previewW = m.width - treeW - 1
		}
		if previewW < 20 {
			previewW = 20
			treeW = m.width - previewW - 1
		}
		ctx.MainContentWidth = treeW
		ctx.PreviewWidth = previewW
		ctx.PreviewHeight = ctx.MainContentHeight
	}

	m.listView.SetSize(ctx.MainContentWidth, ctx.MainContentHeight)
	m.sidePanel.SetSize(ctx.PreviewWidth, ctx.PreviewHeight)
	m.statusBar.SetSize(m.width)
	m.prActivity.SetSize(ctx.MainContentWidth, ctx.MainContentHeight)

	m.listView.syncStyles(&ctx.Styles)
	m.sidePanel.syncStyles(&ctx.Styles)
	m.statusBar.syncStyles(&ctx.Styles)
	m.tabs.syncStyles(&ctx.Styles)
	m.prActivity.syncStyles(&ctx.Styles)
}

func (m *Model) syncProgramContext() {
	m.syncMainContentDimensions()
}
