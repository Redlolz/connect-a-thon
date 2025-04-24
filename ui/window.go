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
	UIComponentButton
	UIComponentInputField
	UIComponentComboBox
)

type UIComponent struct {
	Type   UIComponentType
	W      int32
	H      int32
	Text   string
	Button struct {
		Callback func(*UIWindow)
	}
	InputField struct {
		Identifier string
		Input      string
		texture    *sdl.Texture
	}
	ComboBox struct {
		Identifier  string
		Options     map[int64]string
		OptionsKeys []int64
		selected    int64
		texture     *sdl.Texture
	}
}

type UIWindow struct {
	ui *UI

	X       int32
	Y       int32
	W       int32
	H       int32
	Padding int32

	Center bool

	Components []*UIComponent

	focus    *UIComponent
	focusY   int32
	textArea sdl.Rect

	UserData interface{}
}

func (ui *UI) CreateWindow(x, y, w, h int32) *UIWindow {
	win := UIWindow{
		ui:      ui,
		X:       x,
		Y:       y,
		W:       w,
		H:       h,
		Padding: 4,
		Center:  false,
	}
	return &win
}

func (win *UIWindow) Destroy() {
	for _, c := range win.Components {
		switch c.Type {
		case UIComponentInputField:
			if c.InputField.texture != nil {
				sdl.DestroyTexture(c.InputField.texture)
			}
		case UIComponentComboBox:
			if c.ComboBox.texture != nil {
				sdl.DestroyTexture(c.ComboBox.texture)
			}
		}
	}
}

func (win *UIWindow) SetCenter(state bool) {
	win.Center = state
}

func (win *UIWindow) AddLabel(text string) {
	var w int32
	var h int32

	ttf.GetStringSize(win.ui.Font, text, 0, &w, &h)

	label := UIComponent{
		Type: UIComponentLabel,
		W:    w,
		H:    h,
		Text: text,
	}
	win.Components = append(win.Components, &label)
}

func (win *UIWindow) AddButton(text string, callback func(*UIWindow)) {
	var w int32
	var h int32

	ttf.GetStringSize(win.ui.Font, text, 0, &w, &h)

	button := UIComponent{
		Type: UIComponentButton,
		W:    w,
		H:    h,
		Text: text,
	}
	button.Button.Callback = callback
	win.Components = append(win.Components, &button)
}

func (win *UIWindow) AddInputField(identifier string) *UIComponent {
	h := ttf.GetFontHeight(win.ui.Font)

	inputField := UIComponent{
		Type: UIComponentInputField,
		W:    0,
		H:    h,
	}
	inputField.InputField.Identifier = identifier
	win.Components = append(win.Components, &inputField)

	return &inputField
}

func (win *UIWindow) GetInputField(identifier string) string {
	for _, c := range win.Components {
		if c.Type == UIComponentInputField && c.InputField.Identifier == identifier {
			return c.InputField.Input
		}
	}
	return ""
}

func (win *UIWindow) SetInputField(identifier string, value string) bool {
	for _, c := range win.Components {
		if c.Type == UIComponentInputField && c.InputField.Identifier == identifier {
			c.InputField.Input = value

			surface := ttf.RenderTextBlended(win.ui.Font, c.InputField.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
			c.InputField.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
			sdl.DestroySurface(surface)

			return true
		}
	}
	return false
}

func (win *UIWindow) AddComboBox(identifier string, options map[int64]string) (*UIComponent, error) {
	comboBox := UIComponent{
		Type: UIComponentComboBox,
		W:    0,
		H:    0,
	}
	comboBox.ComboBox.Identifier = identifier
	comboBox.ComboBox.Options = options

	comboBox.ComboBox.OptionsKeys = make([]int64, 0, len(comboBox.ComboBox.Options))
	for k := range comboBox.ComboBox.Options {
		comboBox.ComboBox.OptionsKeys = append(comboBox.ComboBox.OptionsKeys, k)
	}

	// Sort keys
	sort.Slice(comboBox.ComboBox.OptionsKeys, func(i, j int) bool {
		return comboBox.ComboBox.OptionsKeys[i] < comboBox.ComboBox.OptionsKeys[j]
	})

	// Create textures of text
	var textTextures []*sdl.Texture
	var totalHeight int32
	for _, k := range comboBox.ComboBox.OptionsKeys {
		surface := ttf.RenderTextBlended(win.ui.Font, comboBox.ComboBox.Options[k], 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
		texture := sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
		if texture.W > comboBox.W {
			comboBox.W = texture.W
		}
		if texture.H > comboBox.H {
			comboBox.H = texture.H
		}
		totalHeight += texture.H
		textTextures = append(textTextures, texture)
		sdl.DestroySurface(surface)
	}

	comboBox.ComboBox.texture = sdl.CreateTexture(win.ui.Renderer, sdl.PixelFormatRGBA8888, sdl.TextureAccessTarget, comboBox.W, totalHeight)
	sdl.SetRenderTarget(win.ui.Renderer, comboBox.ComboBox.texture)
	sdl.SetRenderDrawColor(win.ui.Renderer, 0, 0, 0, 255)
	sdl.RenderClear(win.ui.Renderer)
	y := float32(0)
	sdl.SetRenderDrawColor(win.ui.Renderer, 255, 255, 255, 255)
	for _, s := range textTextures {
		rect := sdl.FRect{X: 0, Y: y, W: float32(s.W), H: float32(s.H)}
		sdl.RenderTexture(win.ui.Renderer, s, nil, &rect)
		rect.W = float32(comboBox.W)
		sdl.RenderRect(win.ui.Renderer, &rect)
		y += float32(s.H)
	}
	sdl.SetRenderTarget(win.ui.Renderer, nil)

	win.Components = append(win.Components, &comboBox)

	return &comboBox, nil
}

func (win *UIWindow) GetComboBox(identifier string) (int64, error) {
	for _, c := range win.Components {
		if c.Type == UIComponentComboBox && c.ComboBox.Identifier == identifier {
			return c.ComboBox.OptionsKeys[c.ComboBox.selected], nil
		}
	}
	return 0, errors.New("combobox not found")
}

func (win *UIWindow) RenderWindow() {
	x := win.X
	y := win.Y
	w := win.W
	h := win.H

	sdl.RenderDebugTextFormat(win.ui.Renderer, 0, 0, "ui.window: %p", win.ui.window)
	sdl.RenderDebugTextFormat(win.ui.Renderer, 0, 8, "%d", sdl.GetTicksNS())

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

	yOffset := float32(0 + win.Padding)

	for _, c := range win.Components {
		switch c.Type {
		case UIComponentLabel:
			text := ttf.CreateText(win.ui.TextEngine, win.ui.Font, c.Text, 0)
			ttf.DrawRendererText(text, float32(x+win.Padding), float32(y)+yOffset)
			ttf.DestroyText(text)
		case UIComponentButton:
			rect := sdl.FRect{
				X: float32(x + win.Padding),
				Y: float32(y) + yOffset,
				W: float32(c.W),
				H: float32(c.H),
			}
			sdl.RenderRect(win.ui.Renderer, &rect)

			text := ttf.CreateText(win.ui.TextEngine, win.ui.Font, c.Text, 0)
			ttf.DrawRendererText(text, rect.X, rect.Y)
			ttf.DestroyText(text)
		case UIComponentInputField:
			rect := sdl.FRect{
				X: float32(x + win.Padding),
				Y: float32(y) + yOffset,
				W: float32(w - win.Padding*2),
				H: float32(c.H),
			}
			sdl.RenderRect(win.ui.Renderer, &rect)

			if len(c.InputField.Input) > 0 && c.InputField.texture == nil {
				surface := ttf.RenderTextBlended(win.ui.Font, c.InputField.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
				c.InputField.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
				sdl.DestroySurface(surface)
			}

			if c.InputField.texture != nil {
				srcRect := sdl.FRect{
					X: 0,
					Y: 0,
					W: float32(c.InputField.texture.W),
					H: float32(c.InputField.texture.H),
				}
				if c.InputField.texture.W > int32(rect.W) {
					srcRect.W = rect.W
					srcRect.X = float32(c.InputField.texture.W) - rect.W
				}
				dstRect := sdl.FRect{
					X: rect.X,
					Y: rect.Y,
					W: srcRect.W,
					H: srcRect.H,
				}

				sdl.RenderTexture(win.ui.Renderer, c.InputField.texture, &srcRect, &dstRect)
			}
		case UIComponentComboBox:
			srcRect := sdl.FRect{
				X: 0,
				Y: float32(c.ComboBox.selected * int64(c.H)),
				W: float32(c.W),
				H: float32(c.H),
			}
			dstRect := sdl.FRect{
				X: float32(x + win.Padding),
				Y: float32(y) + yOffset,
				W: srcRect.W,
				H: srcRect.H,
			}
			sdl.RenderTexture(win.ui.Renderer, c.ComboBox.texture, &srcRect, &dstRect)

			if win.focus == c {
				sdl.RenderTexture(win.ui.Renderer, c.ComboBox.texture, nil, &sdl.FRect{
					X: float32(x + win.Padding),
					Y: float32(y+c.H) + yOffset,
					W: float32(c.ComboBox.texture.W),
					H: float32(c.ComboBox.texture.H),
				})
			}
		default:
			panic("unknown UIComponent")
		}

		if win.focus == c {
			win.focusY = int32(yOffset)
		}
		yOffset += float32(c.H)
	}

	if win.focus != nil {
		switch win.focus.Type {
		case UIComponentComboBox:
			sdl.RenderTexture(win.ui.Renderer, win.focus.ComboBox.texture, nil, &sdl.FRect{
				X: float32(x + win.Padding),
				Y: float32(y + win.focus.H + win.focusY),
				W: float32(win.focus.ComboBox.texture.W),
				H: float32(win.focus.ComboBox.texture.H),
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
			x1 := win.Padding
			y1 := win.focusY + win.focus.H
			x2 := x1 + win.focus.W
			y2 := y1 + win.focus.ComboBox.texture.H

			if actualX >= x1 && actualX < x2 && actualY >= y1 && actualY < y2 {
				win.focus.ComboBox.selected = int64((actualY - y1) / win.focus.H)
				win.focus = nil
				return
			}
		}
		win.focus = nil
		sdl.StopTextInput(win.ui.Window)
	}

	yOffset := float32(0 + win.Padding)
	for i, c := range win.Components {
		switch c.Type {
		case UIComponentButton:
			x1 := win.Padding
			y1 := int32(yOffset)
			x2 := x1 + c.W
			y2 := y1 + c.H
			if actualX >= x1 && actualX <= x2 && actualY >= y1 && actualY <= y2 {
				c.Button.Callback(win)
				break
			}
		case UIComponentInputField:
			x1 := win.Padding
			y1 := int32(yOffset)
			x2 := x1 + (w - win.Padding*2)
			y2 := y1 + c.H
			if actualX >= x1 && actualX <= x2 && actualY >= y1 && actualY <= y2 {
				win.focus = win.Components[i]
				win.textArea = sdl.Rect{X: x1 + 5, Y: y1, W: (w - win.Padding*2), H: c.H}
				sdl.StartTextInput(win.ui.Window)
				sdl.SetTextInputArea(win.ui.Window, &win.textArea, 1)
				break
			}
		case UIComponentComboBox:
			x1 := win.Padding
			y1 := int32(yOffset)
			x2 := x1 + c.W
			y2 := y1 + c.H
			if actualX >= x1 && actualX <= x2 && actualY >= y1 && actualY <= y2 {
				win.focus = win.Components[i]
				break
			}
		}
		yOffset += float32(c.H)
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
				if len(win.focus.InputField.Input) > 0 {
					_, size := utf8.DecodeLastRuneInString(win.focus.InputField.Input)
					win.focus.InputField.Input = win.focus.InputField.Input[:len(win.focus.InputField.Input)-size]
				}
			default:
				return
			}
			surface := ttf.RenderTextBlended(win.ui.Font, win.focus.InputField.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
			sdl.DestroyTexture(win.focus.InputField.texture)
			win.focus.InputField.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
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
			win.focus.InputField.Input += text
			surface := ttf.RenderTextBlended(win.ui.Font, win.focus.InputField.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
			sdl.DestroyTexture(win.focus.InputField.texture)
			win.focus.InputField.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
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

	yOffset := float32(0 + win.Padding)
	for i, c := range win.Components {
		if c.Type == UIComponentInputField {
			x1 := win.Padding
			y1 := int32(yOffset)
			x2 := x1 + (w - win.Padding*2)
			y2 := y1 + c.H
			if actualX >= x1 && actualX <= x2 && actualY >= y1 && actualY <= y2 {
				win.Components[i].InputField.Input = drop.Data()
				surface := ttf.RenderTextBlended(win.ui.Font, win.Components[i].InputField.Input, 0, sdl.Color{R: 255, G: 255, B: 255, A: 255})
				sdl.DestroyTexture(win.Components[i].InputField.texture)
				win.Components[i].InputField.texture = sdl.CreateTextureFromSurface(win.ui.Renderer, surface)
				sdl.DestroySurface(surface)
			}
		}
		yOffset += float32(c.H)
	}
}
