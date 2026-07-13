package tui

const MinWidth = 80

func splitDetail(width int) bool { return width >= MinWidth }

func treeWidth(width int) int {
	if !splitDetail(width) {
		return width
	}
	return max(width/2, MinWidth/2)
}
