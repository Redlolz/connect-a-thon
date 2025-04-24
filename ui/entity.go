package ui

import (
	"connect-a-thon/conatho"

	"github.com/google/uuid"
	"github.com/jupiterrider/purego-sdl3/img"
	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/jupiterrider/purego-sdl3/ttf"

	_ "embed"
)

type entityMenuItem struct {
	X1 float32
	Y1 float32
	X2 float32
	Y2 float32
}

type MenuItem int

const (
	MenuItemNone MenuItem = iota
	MenuItemSelectImage
	MenuItemEdit
	MenuItemDelete
)

var menuItems = map[MenuItem]string{
	MenuItemSelectImage: "Select Image",
	MenuItemEdit:        "Edit",
	MenuItemDelete:      "Delete",
}

var menuIconTexture *sdl.Texture

func drawMenuIcon(renderer *sdl.Renderer, x, y float32) float32 {
	if menuIconTexture == nil {
		menuIconTexture = sdl.CreateTexture(renderer, sdl.PixelFormatRGBA8888, sdl.TextureAccessTarget, 16, 16)
		sdl.SetRenderTarget(renderer, menuIconTexture)

		sdl.SetRenderDrawColor(renderer, 0, 0, 0, 0)
		sdl.RenderClear(renderer)

		sdl.SetRenderDrawColor(renderer, 255, 255, 255, 255)
		rect := sdl.FRect{X: 1, Y: 0, W: 14, H: 2}
		sdl.RenderFillRect(renderer, &rect)
		rect.Y += 6
		sdl.RenderFillRect(renderer, &rect)
		rect.Y += 6
		sdl.RenderFillRect(renderer, &rect)

		sdl.SetRenderTarget(renderer, nil)
	}

	sdl.RenderTexture(renderer, menuIconTexture, nil, &sdl.FRect{X: x, Y: y, W: float32(menuIconTexture.W), H: float32(menuIconTexture.H)})
	return float32(menuIconTexture.H)
}

//go:embed img/unknown.png
var unknownPng []byte
var unknownTexture *sdl.Texture

func (ui *UI) updateThumbnail(e *conatho.Entity) error {
	thumbnailFile, err := e.EntityGetThumbnail()
	if err != nil {
		return err
	}

	_, ok := ui.ThumbnailCache[e.ID]
	if ok {
		sdl.DestroyTexture(ui.ThumbnailCache[e.ID])
	}

	iostream := sdl.IOFromConstMem(thumbnailFile)
	ui.ThumbnailCache[e.ID] = img.LoadTextureIO(ui.Renderer, iostream, true)

	return nil
}

func (ui *UI) renderThumbnail(e *conatho.Entity, rect *sdl.FRect) {
	if !e.Image {
		if unknownTexture == nil {
			iostream := sdl.IOFromConstMem(unknownPng)
			unknownTexture = img.LoadTextureIO(ui.Renderer, iostream, true)
		}
		sdl.RenderTexture(ui.Renderer, unknownTexture, nil, rect)
	} else {
		thumbnail, ok := ui.ThumbnailCache[e.ID]
		if !ok {
			thumbnailFile, err := e.EntityGetThumbnail()
			if err != nil {
				return
			}

			iostream := sdl.IOFromConstMem(thumbnailFile)
			ui.ThumbnailCache[e.ID] = img.LoadTextureIO(ui.Renderer, iostream, true)
			thumbnail = ui.ThumbnailCache[e.ID]
		}
		sdl.RenderTexture(ui.Renderer, thumbnail, nil, rect)
	}
}

func (ui *UI) RenderEntity(e *conatho.Entity) {
	x := float32(e.X + ui.GlobalX)
	y := float32(e.Y + ui.GlobalY)

	nextY := y + float32(ui.EntityPadding)

	rect := sdl.FRect{X: x, Y: y, W: float32(ui.EntityWidth), H: float32(ui.EntityHeight)}

	sdl.SetRenderDrawColor(ui.Renderer, 0, 0, 0, 255)
	sdl.RenderFillRect(ui.Renderer, &rect)
	sdl.SetRenderDrawColor(ui.Renderer, 255, 255, 255, 255)
	sdl.RenderRect(ui.Renderer, &rect)

	nextY += drawMenuIcon(ui.Renderer, x+float32(ui.EntityPadding), y+float32(ui.EntityPadding)) + float32(ui.EntityPadding)

	imgRect := sdl.FRect{
		X: x + float32((ui.EntityWidth-ui.EntityThumbWidth)/2),
		Y: nextY,
		W: float32(ui.EntityThumbWidth),
		H: float32(ui.EntityThumbHeight),
	}
	ui.renderThumbnail(e, &imgRect)
	sdl.RenderRect(ui.Renderer, &imgRect)
	nextY += imgRect.H

	textName := ttf.CreateText(ui.TextEngine, ui.Font, e.Name, uint64(len(e.Name)))
	defer ttf.DestroyText(textName)
	ttf.DrawRendererText(textName, x+float32(ui.EntityPadding), nextY)
	nextY += float32(ttf.GetFontHeight(ui.Font)) + float32(ui.EntityPadding)

	sdl.RenderDebugTextFormat(ui.Renderer, x+float32(ui.EntityPadding), nextY, "X: %d", e.X)
	nextY += sdl.DebugTextFontCharacterSize + float32(ui.EntityPadding)

	sdl.RenderDebugTextFormat(ui.Renderer, x+4, nextY, "Y: %d", e.Y)

	// Draw handles
	sdl.RenderFillRect(ui.Renderer, &sdl.FRect{
		X: x + float32(ui.EntityWidth/2-(ui.EntityHandleSize/2)),
		Y: y - float32(ui.EntityHandleSize/2),
		W: float32(ui.EntityHandleSize),
		H: float32(ui.EntityHandleSize),
	})
	sdl.RenderFillRect(ui.Renderer, &sdl.FRect{
		X: x + float32(ui.EntityWidth/2-(ui.EntityHandleSize/2)),
		Y: y - float32(ui.EntityHandleSize/2) + float32(ui.EntityHeight),
		W: float32(ui.EntityHandleSize),
		H: float32(ui.EntityHandleSize),
	})
}

func (ui *UI) InEntity(entities map[uuid.UUID]*conatho.Entity, mouseX, mouseY int32) (uuid.UUID, *conatho.Entity) {
	for u, e := range entities {
		if mouseX >= e.X &&
			mouseX <= e.X+ui.EntityWidth &&
			mouseY >= e.Y &&
			mouseY <= e.Y+ui.EntityHeight {
			return u, e
		}
	}
	return uuid.UUID{}, nil
}

func (ui *UI) InEntityMenuButton(entities map[uuid.UUID]*conatho.Entity, mouseX, mouseY int32) (uuid.UUID, *conatho.Entity) {
	for u, e := range entities {
		if mouseX >= e.X+ui.EntityPadding &&
			mouseX <= e.X+ui.EntityPadding+menuIconTexture.W &&
			mouseY >= e.Y+ui.EntityPadding &&
			mouseY <= e.Y+ui.EntityPadding+menuIconTexture.H {
			return u, e
		}
	}
	return uuid.UUID{}, nil
}

func (ui *UI) createEntityMenu() {

	menuWidth := int32(0)
	menuHeight := int32(0)
	w := int32(0)
	h := int32(0)

	for i := MenuItem(1); int(i) <= len(menuItems); i++ {
		ttf.GetStringSize(ui.Font, menuItems[i], 0, &w, &h)
		menuHeight += h
		if w > menuWidth {
			menuWidth = w
		}
	}

	menuWidth += ui.EntityPadding * 2
	menuHeight += ui.EntityPadding * 2

	entityMenu := sdl.CreateTexture(ui.Renderer, sdl.PixelFormatRGBA8888, sdl.TextureAccessTarget, menuWidth, menuHeight)

	sdl.SetRenderTarget(ui.Renderer, entityMenu)
	sdl.SetRenderDrawColor(ui.Renderer, 0, 0, 0, 255)
	sdl.RenderClear(ui.Renderer)

	sdl.SetRenderDrawColor(ui.Renderer, 255, 255, 255, 255)
	sdl.RenderRect(ui.Renderer, &sdl.FRect{X: 0, Y: 0, W: float32(menuWidth), H: float32(menuHeight)})

	nextY := float32(ui.EntityPadding)
	for i := MenuItem(1); int(i) < len(menuItems)+1; i++ {
		renderedText := ttf.CreateText(ui.TextEngine, ui.Font, menuItems[i], 0)
		ttf.DrawRendererText(renderedText, float32(ui.EntityPadding), nextY)
		ttf.DestroyText(renderedText)
		ttf.GetStringSize(ui.Font, menuItems[i], 0, &w, &h)

		if i != 1 {
			sdl.RenderLine(ui.Renderer, 0, nextY, float32(menuWidth), nextY)
		}

		ui.entityMenuItems[i] = entityMenuItem{
			X1: 0,
			Y1: nextY,
			X2: float32(menuWidth),
			Y2: nextY + float32(h),
		}

		nextY += float32(h)
	}

	sdl.SetRenderTarget(ui.Renderer, nil)
	ui.entityMenu = entityMenu
}

func (ui *UI) InEntityMenu(e *conatho.Entity, mouseX, mouseY float32) MenuItem {
	entityX := float32(e.X)
	entityY := float32(e.Y)
	for s, i := range ui.entityMenuItems {
		if mouseX >= entityX+i.X1+float32(ui.EntityPadding)+16 &&
			mouseX <= entityX+i.X2+float32(ui.EntityPadding)+16 &&
			mouseY >= entityY+i.Y1+float32(ui.EntityPadding) &&
			mouseY <= entityY+i.Y2+float32(ui.EntityPadding) {
			return s
		}
	}

	return MenuItemNone
}

func (ui *UI) RenderEntityMenu(e *conatho.Entity) {
	if ui.entityMenu == nil {
		ui.createEntityMenu()
	}
	x := float32(e.X + ui.GlobalX)
	y := float32(e.Y + ui.GlobalY)

	sdl.RenderTexture(ui.Renderer, ui.entityMenu, nil, &sdl.FRect{
		X: x + float32(ui.EntityPadding+16),
		Y: y + float32(ui.EntityPadding),
		W: float32(ui.entityMenu.W),
		H: float32(ui.entityMenu.H),
	})
}

func (ui *UI) InEntitySuperiorHandle(mouseX, mouseY int32) *conatho.Entity {
	for _, e := range ui.Conatho.Entities {
		x1 := e.X + ui.EntityWidth/2 - (ui.EntityHandleSize / 2)
		y1 := e.Y - ui.EntityHandleSize/2 + ui.EntityHeight
		x2 := x1 + ui.EntityHandleSize
		y2 := y1 + ui.EntityHandleSize
		if mouseX >= x1 &&
			mouseX <= x2 &&
			mouseY >= y1 &&
			mouseY <= y2 {
			return e
		}
	}
	return nil
}

func (ui *UI) InEntityInferiorHandle(mouseX, mouseY int32) *conatho.Entity {
	for _, e := range ui.Conatho.Entities {
		x1 := e.X + ui.EntityWidth/2 - (ui.EntityHandleSize / 2)
		y1 := e.Y - ui.EntityHandleSize/2
		x2 := x1 + ui.EntityHandleSize
		y2 := y1 + ui.EntityHandleSize
		if mouseX >= x1 &&
			mouseX <= x2 &&
			mouseY >= y1 &&
			mouseY <= y2 {
			return e
		}
	}
	return nil
}
