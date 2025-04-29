package ui

import (
	"connect-a-thon/conatho"
	"fmt"
	"os"
	"strconv"
	"unsafe"

	"github.com/google/uuid"
	"github.com/jupiterrider/purego-sdl3/img"
	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/jupiterrider/purego-sdl3/ttf"
)

type Action int64

const (
	ActionNone Action = iota
	ActionDragCanvas

	ActionEntityMenu
	ActionDragEntity
	ActionTextField

	ActionConnectionSuperior
	ActionConnectionInferior

	ActionCutConnection

	ActionOpenSubmenu
)

type MenuBarSubMenuItem struct {
	Name     string
	Function func()
}

type MenuBarSubMenu struct {
	Name    string
	Items   []MenuBarSubMenuItem
	X1      int32
	X2      int32
	texture *sdl.Texture
}

type MenuBar struct {
	SubMenus []MenuBarSubMenu
	Height   int32
	Padding  int32
	texture  *sdl.Texture
}

type UI struct {
	GlobalX int32
	GlobalY int32

	EntityWidth       int32
	EntityHeight      int32
	EntityPadding     int32
	EntityHandleSize  int32
	EntityThumbWidth  int32
	EntityThumbHeight int32

	Conatho *conatho.Conatho
	// Entities     map[uuid.UUID]*conatho.Entity
	// EntitiesKeys []uuid.UUID

	Window     *sdl.Window
	Renderer   *sdl.Renderer
	TextEngine *ttf.TextEngine
	Font       *ttf.Font

	ThumbnailCache map[uuid.UUID]*sdl.Texture

	action         Action
	selectedEntity *conatho.Entity
	savedPosX      int32
	savedPosY      int32

	window *UIWindow

	entityMenu      *sdl.Texture
	entityMenuItems map[MenuItem]entityMenuItem

	menuBar            MenuBar
	menuBarOpenSubMenu int
}

func NewUI(window *sdl.Window, renderer *sdl.Renderer, textEngine *ttf.TextEngine, font *ttf.Font) *UI {
	ui := UI{
		GlobalX: 0,
		GlobalY: 0,

		EntityWidth:       150,
		EntityHeight:      200,
		EntityPadding:     4,
		EntityHandleSize:  6,
		EntityThumbWidth:  128,
		EntityThumbHeight: 128,

		Window:         window,
		Renderer:       renderer,
		TextEngine:     textEngine,
		Font:           font,
		ThumbnailCache: make(map[uuid.UUID]*sdl.Texture),

		action: ActionNone,

		entityMenuItems: make(map[MenuItem]entityMenuItem),
	}

	(&ui).MakeMenuBar()

	return &ui
}

func (ui *UI) OpenWindowAdd() {
	ui.CloseWindow()

	addwin := ui.CreateWindow(100, 100, 200, 200)
	addwin.SetCenter(true)

	addwin.AddLabel("Add Entity")
	addwin.AddInputField("name")
	addwin.AddButton("Add", func(win *UIWindow) {
		name := win.GetInputField("name")
		_, err := win.ui.Conatho.CreateEntity(0, 0, name)
		if err != nil {
			panic(err.Error())
		}

		win.SetInputField("name", "")
		win.ui.CloseWindow()
	})
	addwin.AddButton("Close", func(win *UIWindow) {
		win.SetInputField("name", "")
		win.ui.CloseWindow()
	})

	ui.window = addwin
}

func (ui *UI) OpenWindowImageSelect() {
	ui.CloseWindow()

	imgwin := ui.CreateWindow(100, 100, 200, 200)
	imgwin.SetCenter(true)
	imgwin.AddLabel("Select Image")
	imgwin.AddInputField("imagePath")
	imgwin.AddButton("Add", func(win *UIWindow) {
		imagePath := win.GetInputField("imagePath")
		f, err := os.Open(imagePath)
		if err != nil {
			fmt.Println("Could not open file:", err.Error())
			return
		}

		err = win.ui.selectedEntity.EntityAddImage(f)
		if err != nil {
			fmt.Println(err.Error())
		}

		err = win.ui.updateThumbnail(win.ui.selectedEntity)
		if err != nil {
			panic(err.Error())
		}

		win.SetInputField("imagePath", "")
		win.ui.CloseWindow()
	})
	imgwin.AddButton("Close", func(win *UIWindow) {
		win.SetInputField("imagePath", "")
		win.ui.CloseWindow()
	})

	ui.window = imgwin
}

func (ui *UI) OpenWindowEdit(e *conatho.Entity) {
	ui.CloseWindow()

	editwin := ui.CreateWindow(100, 100, 200, 200)
	editwin.SetCenter(true)

	image, err := e.EntityGetImage()
	if err == nil {
		iostream := sdl.IOFromConstMem(image)
		texture := img.LoadTextureIO(ui.Renderer, iostream, true)

		displayWidth := int32(256)
		displayHeight := int32(256)
		if texture.W > texture.H {
			displayHeight = int32(float32(texture.H) / float32(texture.W) * float32(displayWidth))
		} else if texture.H > texture.W {
			displayWidth = int32(float32(texture.W) / float32(texture.H) * float32(displayHeight))
		}

		editwin.AddImage(texture, displayWidth, displayHeight)
	}

	attributes, err := e.GetAttributes()
	if err != nil {
		fmt.Println(err)
	}

	for _, attribute := range attributes {
		editwin.AddLabel(attribute.Name)
		inputFieldComponent := editwin.AddInputField(attribute.Name)

		switch attribute.Type {
		case conatho.DatatypeNumber:
			value := attribute.Number
			inputFieldComponent.Input = strconv.FormatInt(value, 10)
		case conatho.DatatypeString:
			value := attribute.String
			inputFieldComponent.Input = value
		case conatho.DatatypeData:
			value := attribute.Data
			inputFieldComponent.Input = string(value)
		}
	}

	editwin.AddButton("Add", func(win *UIWindow) {
		win.ui.OpenWindowAddAttribute(e)
	})

	editwin.AddButton("Save", func(win *UIWindow) {
		for _, attribute := range attributes {
			newValue := win.GetInputField(attribute.Name)

			switch attribute.Type {
			case conatho.DatatypeNumber:
				value, err := strconv.ParseInt(newValue, 10, 64)
				if err != nil {
					fmt.Println("Could not convert:", value, "to integer")
					continue
				}
				e.UpdateAttribute(attribute.ID, value)
			case conatho.DatatypeString:
				e.UpdateAttribute(attribute.ID, newValue)
			case conatho.DatatypeData:
				e.UpdateAttribute(attribute.ID, []byte(newValue))
			}
		}
		win.ui.CloseWindow()
	})
	editwin.AddButton("Close", func(win *UIWindow) {
		win.ui.CloseWindow()
	})

	ui.window = editwin
}

func (ui *UI) OpenWindowAddAttribute(e *conatho.Entity) {
	ui.CloseWindow()

	attrwin := ui.CreateWindow(100, 100, 200, 200)
	attrwin.SetCenter(true)

	options := make(map[int64]string)
	for k, v := range ui.Conatho.AttributeTypes {
		options[k] = v.Name
	}

	attrwin.AddLabel("Add Attribute")
	attrwin.AddComboBox("combobox", options)

	attrwin.AddButton("Add", func(win *UIWindow) {
		i, err := attrwin.GetComboBox("combobox")
		if err != nil {
			fmt.Println(err)
			return
		}
		e.AddAttribute(i)
		win.ui.OpenWindowEdit(e)
	})

	attrwin.AddButton("Close", func(win *UIWindow) {
		win.ui.OpenWindowEdit(e)
	})

	ui.window = attrwin
}

func (ui *UI) OpenWindowCreateType() {
	ui.CloseWindow()

	attrwin := ui.CreateWindow(100, 100, 200, 200)
	attrwin.SetCenter(true)

	options := make(map[int64]string)
	for k, v := range ui.Conatho.AttributeTypes {
		options[k] = v.Name
	}

	attrwin.AddLabel("Create Type")

	attrwin.AddLabel("Name")
	attrwin.AddInputField("name")

	attrwin.AddLabel("Type")
	attrwin.AddComboBox("type", map[int64]string{
		int64(conatho.DatatypeNumber): "Number",
		int64(conatho.DatatypeString): "String",
		int64(conatho.DatatypeData):   "Data",
	})

	attrwin.AddButton("Add", func(win *UIWindow) {
		name := win.GetInputField("name")

		datatype, err := attrwin.GetComboBox("type")
		if err != nil {
			fmt.Println(err)
			return
		}
		win.ui.Conatho.AddAttributeType(name, conatho.Datatype(datatype))
		win.ui.CloseWindow()
	})

	attrwin.AddButton("Close", func(win *UIWindow) {
		win.ui.CloseWindow()
	})

	ui.window = attrwin
}

func (ui *UI) CloseWindow() {
	if ui.window != nil {
		ui.window.Destroy()
	}
	ui.window = nil
}

func (ui *UI) LoadConatho(fPath string) error {
	con, err := conatho.New(fPath)
	if err != nil {
		return err
	}
	ui.Conatho = &con

	err = con.EntityGetAll()
	if err != nil {
		panic(err.Error())
	}

	err = con.GetAttributeTypes()
	if err != nil {
		panic(err.Error())
	}

	return nil
}

var backgroundTexture *sdl.Texture

func initBackgroundTexture(renderer *sdl.Renderer) *sdl.Texture {
	tex := sdl.CreateTexture(renderer, sdl.PixelFormatRGBA8888, sdl.TextureAccessTarget, 15, 15)
	sdl.SetRenderTarget(renderer, tex)

	sdl.SetRenderDrawColor(renderer, 25, 25, 25, 255)
	sdl.RenderClear(renderer)

	sdl.SetRenderDrawColor(renderer, 60, 60, 60, 255)
	sdl.RenderLine(renderer, 7, 4, 7, 10)
	sdl.RenderLine(renderer, 4, 7, 10, 7)

	sdl.SetRenderTarget(renderer, nil)
	return tex
}

func (c *UI) renderBackground() {
	if backgroundTexture == nil {
		backgroundTexture = initBackgroundTexture(c.Renderer)
	}
	var rendererWidth int32
	var rendererHeight int32
	sdl.GetRenderOutputSize(c.Renderer, &rendererWidth, &rendererHeight)
	dstrect := sdl.FRect{
		X: float32(c.GlobalX%backgroundTexture.W - backgroundTexture.W),
		Y: float32(c.GlobalY%backgroundTexture.H - backgroundTexture.H),
		W: float32(rendererWidth + (backgroundTexture.W * 2)),
		H: float32(rendererHeight + (backgroundTexture.H * 2)),
	}
	// srcrect := sdl.FRect{X: xOffset, Y: yOffset, W: 15, H: 15}
	sdl.RenderTextureTiled(c.Renderer, backgroundTexture, nil, 1.0, &dstrect)
}

func (ui *UI) MakeMenuBar() {
	ui.menuBar = MenuBar{
		SubMenus: []MenuBarSubMenu{
			MenuBarSubMenu{
				Name: "File",
				Items: []MenuBarSubMenuItem{
					MenuBarSubMenuItem{
						Name: "New",
						Function: func() {
							callback := sdl.NewDialogFileCallback(func(userdata unsafe.Pointer, filelist []string, filter int32) {
								if len(filelist) > 0 {
									con, err := conatho.New(filelist[0])
									if err != nil {
										fmt.Println(err)
									} else {
										ui.Conatho = &con
									}
								}
							})
							sdl.NewDialogFileFilter("Conatho Files", "conatho")
							sdl.ShowSaveFileDialog(callback, nil, ui.Window, []sdl.DialogFileFilter{
								sdl.NewDialogFileFilter("Conatho Files", "conatho"),
								sdl.NewDialogFileFilter("All Files", "*"),
							}, "")
						},
					},
					MenuBarSubMenuItem{
						Name: "Open",
						Function: func() {
							callback := sdl.NewDialogFileCallback(func(userdata unsafe.Pointer, filelist []string, filter int32) {
								if len(filelist) > 0 {
									ui.LoadConatho(filelist[0])
								}
							})
							sdl.NewDialogFileFilter("Conatho Files", "conatho")
							sdl.ShowOpenFileDialog(callback, nil, ui.Window, []sdl.DialogFileFilter{
								sdl.NewDialogFileFilter("Conatho Files", "conatho"),
								sdl.NewDialogFileFilter("All Files", "*"),
							}, "", false)
						},
					},
					MenuBarSubMenuItem{
						Name: "Exit",
						Function: func() {
							fmt.Println("Exit")
						},
					},
				},
			},
			MenuBarSubMenu{
				Name: "Attributes",
				Items: []MenuBarSubMenuItem{
					MenuBarSubMenuItem{
						Name: "New Type",
						Function: func() {
							if ui.Conatho != nil {
								ui.OpenWindowCreateType()
							}
						},
					},
				},
			},
		},
		Padding: 5,
	}

	var menuBarTextures []*sdl.Texture
	var menuBarWidth int32
	var menuBarHeight int32

	for i, subMenu := range ui.menuBar.SubMenus {
		var itemTextures []*sdl.Texture
		var subMenuWidth int32
		var subMenuHeight int32
		for _, item := range subMenu.Items {
			surface := ttf.RenderTextBlended(ui.Font, item.Name, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
			itemTextures = append(itemTextures, sdl.CreateTextureFromSurface(ui.Renderer, surface))
			if surface.W+ui.menuBar.Padding*2 > subMenuWidth {
				subMenuWidth = surface.W + ui.menuBar.Padding*2
			}
			subMenuHeight += surface.H + ui.menuBar.Padding*2
			sdl.DestroySurface(surface)
		}

		subMenuTexture := sdl.CreateTexture(ui.Renderer, sdl.PixelFormatRGBA8888,
			sdl.TextureAccessTarget, subMenuWidth, subMenuHeight)
		sdl.SetRenderTarget(ui.Renderer, subMenuTexture)
		sdl.SetRenderDrawColor(ui.Renderer, 0, 0, 0, 255)
		sdl.RenderClear(ui.Renderer)
		y := float32(ui.menuBar.Padding)
		for i, itemTexture := range itemTextures {
			rect := sdl.FRect{
				X: 0,
				Y: float32(i * int(itemTexture.H+ui.menuBar.Padding*2)),
				W: float32(subMenuWidth),
				H: float32(itemTexture.H + ui.menuBar.Padding*2),
			}
			sdl.SetRenderDrawColor(ui.Renderer, 255, 255, 255, 255)
			sdl.RenderRect(ui.Renderer, &rect)
			sdl.RenderTexture(ui.Renderer, itemTexture, nil, &sdl.FRect{
				X: float32(ui.menuBar.Padding),
				Y: rect.Y + float32(ui.menuBar.Padding),
				W: float32(itemTexture.W),
				H: float32(itemTexture.H),
			})
			y += float32(itemTexture.H + ui.menuBar.Padding)
			sdl.DestroyTexture(itemTexture)
		}
		sdl.SetRenderTarget(ui.Renderer, nil)
		ui.menuBar.SubMenus[i].texture = subMenuTexture

		// Menu bar texture
		surface := ttf.RenderTextBlended(ui.Font, subMenu.Name, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
		menuBarTextures = append(menuBarTextures, sdl.CreateTextureFromSurface(ui.Renderer, surface))
		if surface.H > menuBarHeight {
			menuBarHeight = surface.H + ui.menuBar.Padding*2
		}
		menuBarWidth += surface.W + ui.menuBar.Padding*2
		sdl.DestroySurface(surface)
	}

	ui.menuBar.texture = sdl.CreateTexture(ui.Renderer, sdl.PixelFormatRGBA8888, sdl.TextureAccessTarget, menuBarWidth, menuBarHeight)
	sdl.SetRenderTarget(ui.Renderer, ui.menuBar.texture)

	sdl.SetRenderDrawColor(ui.Renderer, 0, 0, 0, 255)
	sdl.RenderClear(ui.Renderer)
	var x int32
	for i, itemTexture := range menuBarTextures {
		rect := sdl.FRect{X: float32(x), Y: 0, W: float32(itemTexture.W + ui.menuBar.Padding*2), H: float32(menuBarHeight)}
		sdl.SetRenderDrawColor(ui.Renderer, 255, 255, 255, 255)
		sdl.RenderRect(ui.Renderer, &rect)
		sdl.RenderTexture(ui.Renderer, itemTexture, nil, &sdl.FRect{
			X: rect.X + float32(ui.menuBar.Padding),
			Y: float32(ui.menuBar.Padding),
			W: float32(itemTexture.W),
			H: float32(itemTexture.H),
		})
		ui.menuBar.SubMenus[i].X1 = x
		ui.menuBar.SubMenus[i].X2 = x + int32(rect.W)
		x += int32(rect.W)
		sdl.DestroyTexture(itemTexture)
	}
	sdl.SetRenderTarget(ui.Renderer, nil)

	ui.menuBar.Height = menuBarHeight
}

func (ui *UI) InMenuBar(mouseX, mouseY int32) (bool, int) {
	if mouseX > 0 && mouseY > 0 && mouseX < ui.menuBar.texture.W && mouseY < ui.menuBar.texture.H {
		for i, submenu := range ui.menuBar.SubMenus {
			if mouseX > submenu.X1 && mouseX < submenu.X2 {
				return true, i

			}
		}
	}
	return false, 0
}

func (ui *UI) InSubMenu(subMenuIndex int, mouseX, mouseY int32) (bool, int) {
	subMenu := ui.menuBar.SubMenus[subMenuIndex]

	x1 := subMenu.X1
	y1 := ui.menuBar.Height
	x2 := x1 + subMenu.texture.W
	y2 := y1 + subMenu.texture.H

	if mouseX > x1 && mouseY > y1 && mouseX < x2 && mouseY < y2 {
		return true, int((mouseY - y1) / (subMenu.texture.H / int32(len(subMenu.Items))))
	}
	return false, 0
}

func (ui *UI) MouseDownMenuBar(button uint8, mouseX, mouseY int32) bool {
	if button == 1 {
		if ui.action == ActionOpenSubmenu {
			ui.action = ActionNone
			if ok, item := ui.InSubMenu(ui.menuBarOpenSubMenu, mouseX, mouseY); ok {
				ui.menuBar.SubMenus[ui.menuBarOpenSubMenu].Items[item].Function()
				return true
			}
		} else {
			if ok, submenu := ui.InMenuBar(mouseX, mouseY); ok {
				ui.menuBarOpenSubMenu = submenu
				ui.action = ActionOpenSubmenu
				return true
			}
		}
	}
	return false
}

func (ui *UI) RenderMenuBar() {
	sdl.RenderTexture(ui.Renderer, ui.menuBar.texture, nil, &sdl.FRect{X: 0, Y: 0, W: float32(ui.menuBar.texture.W), H: float32(ui.menuBar.texture.H)})
	if ui.action == ActionOpenSubmenu {
		subMenu := ui.menuBar.SubMenus[ui.menuBarOpenSubMenu]
		sdl.RenderTexture(ui.Renderer, subMenu.texture, nil, &sdl.FRect{
			X: float32(subMenu.X1),
			Y: float32(ui.menuBar.Height),
			W: float32(subMenu.texture.W),
			H: float32(subMenu.texture.H),
		})
	}
}

func (ui *UI) Render() {
	ui.renderBackground()

	if ui.Conatho != nil {
		ui.RenderCanvas()
	}

	ui.RenderMenuBar()

	if ui.window != nil {
		var rendererWidth int32
		var rendererHeight int32
		sdl.GetRenderOutputSize(ui.Renderer, &rendererWidth, &rendererHeight)
		sdl.SetRenderDrawColor(ui.Renderer, 20, 20, 20, 128)
		sdl.RenderFillRect(ui.Renderer, &sdl.FRect{X: 0, Y: 0, W: float32(rendererWidth), H: float32(rendererHeight)})

		ui.window.RenderWindow()
	}
}

func (ui *UI) MouseDown(button uint8, mouseX, mouseY int32) {
	if ui.MouseDownMenuBar(button, mouseX, mouseY) {
		return
	}

	if ui.window == nil && ui.Conatho != nil {
		ui.MouseDownCanvas(button, mouseX, mouseY)
	} else if ui.window != nil {
		ui.window.MouseDown(button, mouseX, mouseY)
	}
}

func (ui *UI) MouseUp(button uint8, mouseX, mouseY int32) {
	if ui.window == nil && ui.Conatho != nil {
		ui.MouseUpCanvas(button, mouseX, mouseY)
	}
}

func (ui *UI) MouseMotion(relX, relY float32) {
	ui.MouseMotionCanvas(relX, relY)
}

func (ui *UI) MouseWheel(direction sdl.MouseWheelDirection, x, y, mouseX, mouseY int32) {
	if ui.window != nil {
		ui.window.MouseWheel(direction, x, y, mouseX, mouseY)
	}
}

func (ui *UI) KeyDown(key sdl.Keycode) {
	if ui.window == nil && ui.Conatho != nil {
		ui.KeyDownCanvas(key)
	} else if ui.window != nil {
		ui.window.KeyDown(key)
	}
}

func (ui *UI) TextInput(text string) {
	if ui.window != nil {
		ui.window.TextInput(text)
	}
}

func (ui *UI) Drop(drop sdl.DropEvent) {
	if ui.window != nil {
		ui.window.Drop(drop)
	}
}
