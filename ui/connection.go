package ui

import (
	"connect-a-thon/conatho"

	"github.com/google/uuid"
	"github.com/jupiterrider/purego-sdl3/sdl"
)

func onSegment(x1, y1, x2, y2, x3, y3 int32) bool {
	if x2 <= max(x1, x3) && x2 >= min(x1, x3) &&
		y2 <= max(y1, y3) && y2 >= min(y1, y3) {
		return true
	}

	return false
}

// To find orientation of ordered triplet (p, q, r).
// The function returns following values
// 0 --> p, q and r are collinear
// 1 --> Clockwise
// 2 --> Counterclockwise
func orientation(x1, y1, x2, y2, x3, y3 int32) int32 {
	// (Point p, Point q, Point r)
	// See https://www.geeksforgeeks.org/orientation-3-ordered-points/
	// for details of below formula.
	val := (y2-y1)*(x3-x2) - (x2-x1)*(y3-y2)

	if val == 0 {
		return 0 // collinear
	}

	// clock or counterclock wise
	if val > 0 {
		return 1
	} else {
		return 2
	}
}

// The main function that returns true if line segment 'p1q1'
// and 'p2q2' intersect.
func doIntersect(x1, y1, x2, y2, x3, y3, x4, y4 int32) bool {
	// (Point p1, Point q1, Point p2, Point q2)
	// Find the four orientations needed for general and
	// special cases
	o1 := orientation(x1, y1, x2, y2, x3, y3)
	o2 := orientation(x1, y1, x2, y2, x4, y4)
	o3 := orientation(x3, y3, x4, y4, x1, y1)
	o4 := orientation(x3, y3, x4, y4, x2, y2)

	// General case
	if o1 != o2 && o3 != o4 {
		return true
	}

	// Special Cases
	// p1, q1 and p2 are collinear and p2 lies on segment p1q1
	if o1 == 0 && onSegment(x1, y1, x3, y3, x2, y2) {
		return true
	}

	// p1, q1 and q2 are collinear and q2 lies on segment p1q1
	if o2 == 0 && onSegment(x1, y1, x4, y4, x2, y2) {
		return true
	}

	// p2, q2 and p1 are collinear and p1 lies on segment p2q2
	if o3 == 0 && onSegment(x3, y3, x1, y1, x4, y4) {
		return true
	}

	// p2, q2 and q1 are collinear and q1 lies on segment p2q2
	if o4 == 0 && onSegment(x3, y3, x2, y2, x4, y4) {
		return true
	}

	return false // Doesn't fall in any of the above cases
}

func (ui *UI) RenderConnection(connectionID uuid.UUID) {
	connection := ui.Conatho.Connections[connectionID]

	superior := ui.Conatho.Entities[connection.Superior]
	inferior := ui.Conatho.Entities[connection.Inferior]

	superiorConX := float32((superior.X + ui.EntityWidth/2) + ui.GlobalX)
	superiorConY := float32((superior.Y + ui.EntityHeight) + ui.GlobalY)

	inferiorConX := float32((inferior.X + ui.EntityWidth/2) + ui.GlobalX)
	inferiorConY := float32(inferior.Y + ui.GlobalY)

	sdl.RenderLine(ui.Renderer, superiorConX, superiorConY, inferiorConX, inferiorConY)
}

func (ui *UI) CrossesConnection(x1, y1, x2, y2 int32) *conatho.Connection {
	for _, k := range ui.Conatho.ConnectionsKeys {
		connection := ui.Conatho.Connections[k]

		superior := ui.Conatho.Entities[connection.Superior]
		inferior := ui.Conatho.Entities[connection.Inferior]

		superiorConX := (superior.X + ui.EntityWidth/2)
		superiorConY := (superior.Y + ui.EntityHeight)

		inferiorConX := (inferior.X + ui.EntityWidth/2)
		inferiorConY := inferior.Y

		if doIntersect(x1, y1, x2, y2, superiorConX, superiorConY, inferiorConX, inferiorConY) {
			return connection
		}
	}
	return nil
}
