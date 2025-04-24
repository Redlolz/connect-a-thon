package ui

import (
	"fmt"

	"github.com/jupiterrider/purego-sdl3/sdl"
)

func (ui *UI) RenderCanvas() {
	sdl.SetRenderDrawColor(ui.Renderer, 255, 255, 255, 255)

	for _, k := range ui.Conatho.ConnectionsKeys {
		ui.RenderConnection(k)
	}

	for _, k := range ui.Conatho.EntitiesKeys {
		ui.RenderEntity(ui.Conatho.Entities[k])
		if ui.action == ActionEntityMenu && ui.selectedEntity == ui.Conatho.Entities[k] {
			ui.RenderEntityMenu(ui.Conatho.Entities[k])
		}
	}

	// Render line to cursor if action is active
	if ui.action == ActionConnectionSuperior || ui.action == ActionConnectionInferior {
		x1 := float32((ui.selectedEntity.X + ui.EntityWidth/2) + ui.GlobalX)
		y1 := float32(0)
		if ui.action == ActionConnectionSuperior {
			y1 = float32((ui.selectedEntity.Y + ui.EntityHeight) + ui.GlobalY)
		} else {
			y1 = float32(ui.selectedEntity.Y + ui.GlobalY)
		}
		var x2 float32
		var y2 float32
		sdl.GetMouseState(&x2, &y2)

		sdl.SetRenderDrawColor(ui.Renderer, 255, 255, 255, 255)
		sdl.RenderLine(ui.Renderer, x1, y1, x2, y2)
	}

	if ui.action == ActionCutConnection {
		var x2 float32
		var y2 float32
		sdl.GetMouseState(&x2, &y2)
		sdl.SetRenderDrawColor(ui.Renderer, 255, 0, 0, 255)
		sdl.RenderLine(ui.Renderer, float32(ui.savedPosX+ui.GlobalX), float32(ui.savedPosY+ui.GlobalY), x2, y2)
	}
}

func (ui *UI) MouseDownCanvas(button uint8, mouseX, mouseY int32) {
	actualX := mouseX - ui.GlobalX
	actualY := mouseY - ui.GlobalY

	if button == 1 && ui.action == ActionEntityMenu {
		thing := ui.InEntityMenu(ui.selectedEntity, float32(actualX), float32(actualY))
		switch thing {
		case MenuItemEdit:
			ui.action = ActionNone
			ui.OpenWindowEdit(ui.selectedEntity)
		case MenuItemSelectImage:
			ui.action = ActionNone
			ui.OpenWindowImageSelect()
		case MenuItemDelete:
			err := ui.selectedEntity.Delete()
			if err != nil {
				panic(err.Error())
			}
		default:
			ui.action = ActionNone
		}
	} else if button == 1 {
		ui.action = ActionNone

		entity := ui.InEntitySuperiorHandle(actualX, actualY)
		if entity != nil {
			ui.action = ActionConnectionSuperior
			ui.selectedEntity = entity
			return
		}

		entity = ui.InEntityInferiorHandle(actualX, actualY)
		if entity != nil {
			ui.action = ActionConnectionInferior
			ui.selectedEntity = entity
			return
		}

		_, entity = ui.InEntityMenuButton(ui.Conatho.Entities, actualX, actualY)
		if entity != nil {
			ui.action = ActionEntityMenu
			ui.selectedEntity = entity
			return
		}

		if ui.action == ActionNone {
			ui.action = ActionCutConnection
			ui.savedPosX = actualX
			ui.savedPosY = actualY
		}
	} else if button == 3 {
		_, entity := ui.InEntity(ui.Conatho.Entities, actualX, actualY)
		if entity != nil {
			ui.action = ActionDragEntity
			ui.selectedEntity = entity
		} else {
			ui.action = ActionDragCanvas
		}
	}
}

func (ui *UI) MouseUpCanvas(button uint8, mouseX, mouseY int32) {
	actualX := mouseX - ui.GlobalX
	actualY := mouseY - ui.GlobalY

	if button == 1 {
		if ui.action == ActionConnectionSuperior {
			entity := ui.InEntityInferiorHandle(actualX, actualY)
			if entity != nil {
				err := ui.selectedEntity.ConnectTo(entity, "")
				if err != nil {
					fmt.Println(err)
				}
			}
			ui.action = ActionNone
		} else if ui.action == ActionConnectionInferior {
			entity := ui.InEntitySuperiorHandle(actualX, actualY)
			if entity != nil {
				err := entity.ConnectTo(ui.selectedEntity, "")
				if err != nil {
					fmt.Println(err)
				}
			}
			ui.action = ActionNone
		} else if ui.action == ActionCutConnection {
			// Check and cut connections
			connection := ui.CrossesConnection(ui.savedPosX, ui.savedPosY, actualX, actualY)
			if connection != nil {
				ui.Conatho.RemoveConnection(connection)
			}
			ui.action = ActionNone
		}
	} else if button == 3 {
		if ui.action == ActionDragEntity {
			err := ui.selectedEntity.UpdatePosition()
			if err != nil {
				panic("Could not update position for entity")
			}
			ui.selectedEntity = nil
		}
		ui.action = ActionNone
	}
}

func (ui *UI) MouseMotionCanvas(relX, relY float32) {
	if ui.action == ActionDragCanvas {
		ui.GlobalX += int32(relX)
		ui.GlobalY += int32(relY)
	} else if ui.action == ActionDragEntity {
		ui.selectedEntity.X += int32(relX)
		ui.selectedEntity.Y += int32(relY)
	}
}

func (ui *UI) KeyDownCanvas(key sdl.Keycode) {
	switch key {
	case sdl.KeycodeA:
		ui.OpenWindowAdd()
	case sdl.KeycodeEscape:
		ui.CloseWindow()
	}
}
