package ui

import (
	"errors"
	"sort"
	"unicode/utf8"

	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/jupiterrider/purego-sdl3/ttf"
)

type UIComponentType int64

const (
	UIComponentLabel UIComponentType = iota
	UIComponentImage
	UIComponentButton
	UIComponentInputField
	UIComponentComboBox
)

type UIComponent struct {
	Type UIComponentType

	x int32
	y int32
	w int32
	h int32

	Text       string
	Identifier string
	texture    *sdl.Texture

	// 	UIComponentButton
	Callback func(*UIWindow)
	// UIComponentInputField
	Input string
	//UIComponentComboBox
	Options     map[int64]string
	OptionsKeys []int64
	selected    int64
}

type hitbox struct {
	component *UIComponent
	x1        int32
	y1        int32
	x2        int32
	y2        int32
}

type UIWindow struct {
	ui *UI

	X int32
	Y int32
	W int32
	H int32

	WindowPadding    int32
	ComponentPadding int32
	ComponentMargin  int32

	AutoWidth  bool
	AutoHeight bool
	Center     bool

	Components []*UIComponent
	hitboxes   []hitbox

	focus    *UIComponent
	focusY   int32
	textArea sdl.Rect

	UserData interface{}
}

func (ui *UI) CreateWindow(x, y, w, h int32) *UIWindow {
	win := UIWindow{
		ui:              ui,
		X:               x,
		Y:               y,
		W:               w,
		H:               h,
		WindowPadding:   4,
		Center:          false,
		AutoWidth:       true,
		AutoHeight:      true,
		ComponentMargin: 4,
	}
	return &win
}

func (win *UIWindow) Destroy() {
	for _, c := range win.Components {
		if c.texture != nil {
			sdl.DestroyTexture(c.texture)
		}
	}
}

func (win *UIWindow) SetCenter(state bool) {
	win.Center = state
}

func (win *UIWindow) nextX() int32 {
	if len(win.Components) == 0 {
		return win.WindowPadding
	}

	lastComponent := win.Components[len(win.Components)-1]
	if lastComponent.Type == UIComponentButton {
		return lastComponent.x + lastComponent.w + win.ComponentMargin
	}
	return win.WindowPadding
}

func (win *UIWindow) nextY() int32 {
	if len(win.Components) == 0 {
		return win.WindowPadding + win.ComponentMargin
	}

	lastComponent := win.Components[len(win.Components)-1]
	if lastComponent.Type == UIComponentButton {
		return lastComponent.y
	}
	return lastComponent.y + lastComponent.h + win.ComponentMargin
}

func (win *UIWindow) addComponent(component *UIComponent) {
	if win.AutoWidth {
		if len(win.Components) == 0 {
			win.W = 0
		}
		if win.WindowPadding*2+component.x+component.w > win.W {
			win.W = component.x + component.w + win.WindowPadding*2
		}
	}
	if win.AutoHeight {
		if len(win.Components) == 0 {
			win.H = 0
		}
		win.H += component.h + win.ComponentMargin/2
	}

	if component.Type != UIComponentLabel {
		win.hitboxes = append(win.hitboxes, hitbox{
			component: component,
			x1:        component.x,
			y1:        component.y,
			x2:        component.x + component.w,
			y2:        component.y + component.h,
		})
	}

	win.Components = append(win.Components, component)
}

func (win *UIWindow) AddLabel(text string) {
	var w int32
	var h int32

	ttf.GetStringSize(win.ui.Font, text, 0, &w, &h)

	label := UIComponent{
		Type: UIComponentLabel,
		w:    w,
		h:    h,
		x:    win.nextX(),
		y:    win.nextY(),
		Text: text,
	}

	win.addComponent(&label)
}

func (win *UIWindow) AddImage(img *sdl.Texture, w, h int32) {
	image := UIComponent{
		Type:    UIComponentImage,
		w:       w,
		h:       h,
		x:       win.nextX(),
		y:       win.nextY(),
		texture: img,
	}

	win.addComponent(&image)
}

func (win *UIWindow) AddButton(text string, callback func(*UIWindow)) {
	var w int32
	var h int32

	ttf.GetStringSize(win.ui.Font, text, 0, &w, &h)

	button := UIComponent{
		Type:     UIComponentButton,
		w:        w,
		h:        h,
		x:        win.nextX(),
		y:        win.nextY(),
		Text:     text,
		Callback: callback,
	}

	win.addComponent(&button)
}

func (win *UIWindow) AddInputField(identifier string) *UIComponent {
	h := ttf.GetFontHeight(win.ui.Font)

	inputField := UIComponent{
		Type:       UIComponentInputField,
		w:          200,
		h:          h,
		x:          win.nextX(),
		y:          win.nextY(),
		Identifier: identifier,
	}

	win.addComponent(&inputField)

	return &inputField
}

func (win *UIWindow) GetInputField(identifier string) string {
	for _, c := range win.Components {
		if c.Type == UIComponentInputField && c.Identifier == identifier {
			return c.Input
		}
	}
	return ""
}

func (win *UIWindow) SetInputField(identifier string, value string) bool {
	for _, c := range win.Components {
		if c.Type == UIComponentInputField && c.Identifier == identifier {
			c.Input = value

			surface := ttf.RenderTextBlended(win.ui.Font, c.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
			c.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
			sdl.DestroySurface(surface)

			return true
		}
	}
	return false
}

func (win *UIWindow) AddComboBox(identifier string, options map[int64]string) (*UIComponent, error) {
	comboBox := UIComponent{
		Type:       UIComponentComboBox,
		w:          0,
		h:          ttf.GetFontHeight(win.ui.Font),
		x:          win.nextX(),
		y:          win.nextY(),
		Identifier: identifier,
		Options:    options,
	}

	comboBox.OptionsKeys = make([]int64, 0, len(comboBox.Options))
	for k := range comboBox.Options {
		comboBox.OptionsKeys = append(comboBox.OptionsKeys, k)
	}

	// Sort keys
	sort.Slice(comboBox.OptionsKeys, func(i, j int) bool {
		return comboBox.OptionsKeys[i] < comboBox.OptionsKeys[j]
	})

	// Create textures of text
	var textTextures []*sdl.Texture
	var totalHeight int32

	for _, k := range comboBox.OptionsKeys {
		surface := ttf.RenderTextBlended(win.ui.Font, comboBox.Options[k], 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
		if surface != nil {
			texture := sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
			if texture.W > comboBox.w {
				comboBox.w = texture.W
			}
			textTextures = append(textTextures, texture)
			sdl.DestroySurface(surface)
		} else {
			textTextures = append(textTextures, nil)
		}
		totalHeight += ttf.GetFontHeight(win.ui.Font)
	}

	comboBox.texture = sdl.CreateTexture(win.ui.Renderer, sdl.PixelFormatRGBA8888, sdl.TextureAccessTarget, comboBox.w, totalHeight)
	sdl.SetRenderTarget(win.ui.Renderer, comboBox.texture)
	sdl.SetRenderDrawColor(win.ui.Renderer, 0, 0, 0, 255)
	sdl.RenderClear(win.ui.Renderer)
	y := float32(0)
	sdl.SetRenderDrawColor(win.ui.Renderer, 255, 255, 255, 255)
	for _, s := range textTextures {
		if s != nil {
			rect := sdl.FRect{X: 0, Y: y, W: float32(s.W), H: float32(s.H)}
			sdl.RenderTexture(win.ui.Renderer, s, nil, &rect)
			rect.W = float32(comboBox.w)
			sdl.RenderRect(win.ui.Renderer, &rect)
		} else {
			rect := sdl.FRect{X: 0, Y: y, W: float32(comboBox.w), H: float32(ttf.GetFontHeight(win.ui.Font))}
			rect.W = float32(comboBox.w)
			sdl.RenderRect(win.ui.Renderer, &rect)
		}
		y += float32(ttf.GetFontHeight(win.ui.Font))
	}
	sdl.SetRenderTarget(win.ui.Renderer, nil)

	win.addComponent(&comboBox)
	return &comboBox, nil
}

func (win *UIWindow) GetComboBox(identifier string) (int64, error) {
	for _, c := range win.Components {
		if c.Type == UIComponentComboBox && c.Identifier == identifier {
			return c.OptionsKeys[c.selected], nil
		}
	}
	return 0, errors.New("combobox not found")
}

func (win *UIWindow) RenderWindow() {
	x := win.X
	y := win.Y
	w := win.W
	h := win.H

	if win.Center {
		var rendererW int32
		var rendererH int32
		sdl.GetRenderOutputSize(win.ui.Renderer, &rendererW, &rendererH)
		x = rendererW/2 - (w / 2)
		y = rendererH/2 - (h / 2)
	}
	winRect := sdl.FRect{X: float32(x), Y: float32(y), W: float32(w), H: float32(h)}

	sdl.SetRenderDrawColor(win.ui.Renderer, 0, 0, 0, 255)
	sdl.RenderFillRect(win.ui.Renderer, &winRect)

	sdl.SetRenderDrawColor(win.ui.Renderer, 255, 255, 255, 255)
	sdl.RenderRect(win.ui.Renderer, &winRect)

	for _, c := range win.Components {
		switch c.Type {
		case UIComponentLabel:
			text := ttf.CreateText(win.ui.TextEngine, win.ui.Font, c.Text, 0)
			ttf.DrawRendererText(text, float32(x+c.x), float32(y+c.y))
			ttf.DestroyText(text)
		case UIComponentImage:
			sdl.RenderTexture(win.ui.Renderer, c.texture, nil, &sdl.FRect{
				X: float32(x + c.x),
				Y: float32(y + c.y),
				W: float32(c.w),
				H: float32(c.h),
			})
		case UIComponentButton:
			rect := sdl.FRect{
				X: float32(x + c.x),
				Y: float32(y + c.y),
				W: float32(c.w),
				H: float32(c.h),
			}
			sdl.RenderRect(win.ui.Renderer, &rect)

			text := ttf.CreateText(win.ui.TextEngine, win.ui.Font, c.Text, 0)
			ttf.DrawRendererText(text, rect.X, rect.Y)
			ttf.DestroyText(text)
		case UIComponentInputField:
			rect := sdl.FRect{
				X: float32(x + c.x),
				Y: float32(y + c.y),
				W: float32(c.w),
				H: float32(c.h),
			}
			sdl.RenderRect(win.ui.Renderer, &rect)

			if len(c.Input) > 0 && c.texture == nil {
				surface := ttf.RenderTextBlended(win.ui.Font, c.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
				c.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
				sdl.DestroySurface(surface)
			}

			if c.texture != nil {
				srcRect := sdl.FRect{
					X: 0,
					Y: 0,
					W: float32(c.texture.W),
					H: float32(c.texture.H),
				}
				if c.texture.W > int32(rect.W) {
					srcRect.W = rect.W
					srcRect.X = float32(c.texture.W) - rect.W
				}
				dstRect := sdl.FRect{
					X: rect.X,
					Y: rect.Y,
					W: srcRect.W,
					H: srcRect.H,
				}

				sdl.RenderTexture(win.ui.Renderer, c.texture, &srcRect, &dstRect)
			}
		case UIComponentComboBox:
			srcRect := sdl.FRect{
				X: 0,
				Y: float32(c.selected * int64(c.h)),
				W: float32(c.w),
				H: float32(c.h),
			}
			dstRect := sdl.FRect{
				X: float32(x + c.x),
				Y: float32(y + c.y),
				W: float32(c.w),
				H: float32(c.h),
			}
			sdl.RenderTexture(win.ui.Renderer, c.texture, &srcRect, &dstRect)
		default:
			panic("unknown UIComponent")
		}
	}

	if win.focus != nil {
		switch win.focus.Type {
		case UIComponentComboBox:
			sdl.RenderTexture(win.ui.Renderer, win.focus.texture, nil, &sdl.FRect{
				X: float32(x + win.focus.x),
				Y: float32(y + win.focus.y + win.focus.h),
				W: float32(win.focus.texture.W),
				H: float32(win.focus.texture.H),
			})
		}
	}
}

func (win *UIWindow) MouseDown(button uint8, mouseX, mouseY int32) {
	x := win.X
	y := win.Y
	w := win.W
	h := win.H

	if win.Center {
		var rendererW int32
		var rendererH int32
		sdl.GetRenderOutputSize(win.ui.Renderer, &rendererW, &rendererH)
		x = rendererW/2 - (w / 2)
		y = rendererH/2 - (h / 2)
	}

	actualX := mouseX - x
	actualY := mouseY - y

	if win.focus != nil {
		switch win.focus.Type {
		case UIComponentComboBox:
			x1 := win.focus.x
			y1 := win.focus.y + win.focus.h
			x2 := x1 + win.focus.w
			y2 := y1 + win.focus.texture.H

			if actualX >= x1 && actualX < x2 && actualY >= y1 && actualY < y2 {
				win.focus.selected = int64((actualY - y1) / win.focus.h)
				win.focus = nil
				return
			}
		}
		win.focus = nil
		sdl.StopTextInput(win.ui.Window)
	}

	for _, c := range win.hitboxes {
		if actualX >= c.x1 && actualX <= c.x2 && actualY >= c.y1 && actualY <= c.y2 {
			switch c.component.Type {
			case UIComponentButton:
				c.component.Callback(win)
			case UIComponentInputField:
				win.focus = c.component
				sdl.StartTextInput(win.ui.Window)
				// win.textArea = sdl.Rect{X: x1 + 5, Y: y1, W: (w - win.WindowPadding*2), H: c.h}
				// sdl.SetTextInputArea(win.ui.Window, &win.textArea, 1)
			case UIComponentComboBox:
				win.focus = c.component
			}
			break
		}
	}
}

func (win *UIWindow) KeyDown(key sdl.Keycode) {
	if win.focus != nil {
		if win.focus.Type == UIComponentInputField {
			// sdl.SetTextInputArea(win.ui.Window, &win.textArea, 0)
			switch key {
			case sdl.KeycodeEscape:
				win.focus = nil
				sdl.StopTextInput(win.ui.Window)
				return
			case sdl.KeycodeBackspace:
				// Removes last unicode character
				if len(win.focus.Input) > 0 {
					_, size := utf8.DecodeLastRuneInString(win.focus.Input)
					win.focus.Input = win.focus.Input[:len(win.focus.Input)-size]
				}
			default:
				return
			}
			surface := ttf.RenderTextBlended(win.ui.Font, win.focus.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
			sdl.DestroyTexture(win.focus.texture)
			win.focus.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
			sdl.DestroySurface(surface)
		}
	} else {
		switch key {
		case sdl.KeycodeEscape:
			win.ui.CloseWindow()
		}
	}
}

func (win *UIWindow) TextInput(text string) {
	if win.focus != nil {
		if win.focus.Type == UIComponentInputField {
			win.focus.Input += text
			surface := ttf.RenderTextBlended(win.ui.Font, win.focus.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
			sdl.DestroyTexture(win.focus.texture)
			win.focus.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
			sdl.DestroySurface(surface)
		}
	}
}

func (win *UIWindow) Drop(drop sdl.DropEvent) {
	x := win.X
	y := win.Y
	w := win.W
	h := win.H

	if win.Center {
		var rendererW int32
		var rendererH int32
		sdl.GetRenderOutputSize(win.ui.Renderer, &rendererW, &rendererH)
		x = rendererW/2 - (w / 2)
		y = rendererH/2 - (h / 2)
	}

	actualX := int32(drop.X) - x
	actualY := int32(drop.Y) - y

	for _, c := range win.hitboxes {
		if actualX >= c.x1 && actualX <= c.x2 && actualY >= c.y1 && actualY <= c.y2 {
			if c.component.Type == UIComponentInputField {
				c.component.Input = drop.Data()
				surface := ttf.RenderTextBlended(win.ui.Font, c.component.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
				sdl.DestroyTexture(c.component.texture)
				c.component.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
				sdl.DestroySurface(surface)
			}
		}
	}
}
