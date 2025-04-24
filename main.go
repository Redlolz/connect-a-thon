package main

import (
	"connect-a-thon/conatho"
	"connect-a-thon/ui"

	_ "net/http/pprof"

	"fmt"

	"github.com/jupiterrider/purego-sdl3/sdl"
	"github.com/jupiterrider/purego-sdl3/ttf"
)

var con conatho.Conatho

func main() {
	// go func() {
	// 	log.Println(http.ListenAndServe("localhost:6060", nil))
	// }()

	if !sdl.SetHint(sdl.HintRenderVSync, "1") {
		panic(sdl.GetError())
	}

	defer sdl.Quit()
	if !sdl.Init(sdl.InitVideo) {
		panic(sdl.GetError())
	}

	var window *sdl.Window
	var renderer *sdl.Renderer
	if !sdl.CreateWindowAndRenderer("Connect-A-Thon", 640, 480, sdl.WindowResizable, &window, &renderer) {
		panic(sdl.GetError())
	}

	sdl.SetRenderDrawBlendMode(renderer, sdl.BlendModeBlend)

	if !ttf.Init() {
		panic(sdl.GetError())
	}
	textengine := ttf.CreateRendererTextEngine(renderer)
	if textengine == nil {
		panic(sdl.GetError())
	}

	defer ttf.DestroyRendererTextEngine(textengine)
	defer sdl.DestroyRenderer(renderer)
	defer sdl.DestroyWindow(window)

	font := ttf.OpenFont("assets/overpass-mono/OverpassMono-Bold.ttf", 14)
	if font == nil {
		fmt.Println("Font could not be loaded!")
		fmt.Println(sdl.GetError())
	}

	// var entities []Entity
	// entities = append(entities, Entity{
	// 	ID:   uuid.New(),
	// 	X:    100,
	// 	Y:    100,
	// 	Name: "Example Entity 1",
	// })

	// entities = append(entities, Entity{
	// 	ID:   uuid.New(),
	// 	X:    300,
	// 	Y:    100,
	// 	Name: "Example Entity 2",
	// })

	// e1, err := con.CreateEntity(100, 100, "Test Entity 1")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	// e2, err := con.CreateEntity(100, 100, "Test Entity 2")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	// e3, err := con.CreateEntity(100, 100, "Test Entity 3")
	// if err != nil {
	// 	fmt.Println(err.Error())
	// }

	// e1.EntityConnectTo(&e2, "Employs")
	// e1.EntityConnectTo(&e3, "Employs")

	ui := ui.NewUI(window, renderer, textengine, font)
	// ui.LoadConatho("test.conatho")

	running := true
	for running {
		var event sdl.Event

		for sdl.PollEvent(&event) {
			switch event.Type() {
			case sdl.EventQuit:
				running = false
			case sdl.EventMouseButtonDown:
				ui.MouseDown(event.Button().Button, int32(event.Motion().X), int32(event.Motion().Y))
			case sdl.EventMouseButtonUp:
				ui.MouseUp(event.Button().Button, int32(event.Motion().X), int32(event.Motion().Y))
			case sdl.EventMouseMotion:
				ui.MouseMotion(event.Motion().Xrel, event.Motion().Yrel)
			case sdl.EventKeyDown:
				ui.KeyDown(event.Key().Key)
			case sdl.EventTextInput:
				input := event.Text()
				ui.TextInput(input.Text())
			case sdl.EventTextEditingCandidates:
				fmt.Println("FIXME")
			case sdl.EventTextEditing:
				fmt.Println("FIXME")
			case sdl.EventDropFile:
				drop := event.Drop()
				ui.Drop(drop)
				fmt.Println(drop.Data(), drop.Source())
			case sdl.EventDropText:
				drop := event.Drop()
				fmt.Println(drop.Data(), drop.Source())
			}
		}
		sdl.SetRenderDrawColor(renderer, 0, 0, 0, 255)
		sdl.RenderClear(renderer)

		ui.Render()

		sdl.RenderPresent(renderer)
	}
}
