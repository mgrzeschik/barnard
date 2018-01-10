package uiterm // import "layeh.com/barnard/uiterm"

import (
	"errors"
	"sync/atomic"

	"github.com/nsf/termbox-go"
)

type KeyListener func(ui *Ui, key Key)

type UiManager interface {
	OnUiInitialize()
	OnUiResize(ui *Ui, width, height int)
}

type Ui struct {
	Fg, Bg Attribute

	close   chan bool
	manager UiManager

	drawCount     int32
	elements      map[string]*uiElement
	activeElement *uiElement

	keyListeners map[Key][]KeyListener
}

type uiElement struct {
	Name           string
	X0, Y0, X1, Y1 int
	View           View
}

func New(manager UiManager) *Ui {
	ui := &Ui{
		close:        make(chan bool, 10),
		elements:     make(map[string]*uiElement),
		manager:      manager,
		keyListeners: make(map[Key][]KeyListener),
	}
	return ui
}

func (ui *Ui) Close() {
	if termbox.IsInit {
		ui.close <- true
	}
}

func (ui *Ui) Refresh() {
	if termbox.IsInit {
		ui.beginDraw()
		defer ui.endDraw()

		termbox.Clear(termbox.Attribute(ui.Fg), termbox.Attribute(ui.Bg))
		termbox.HideCursor()
		for _, element := range ui.elements {
			element.View.uiDraw()
		}
	}
}

func (ui *Ui) beginDraw() {
	atomic.AddInt32(&ui.drawCount, 1)
}

func (ui *Ui) endDraw() {
	if count := atomic.AddInt32(&ui.drawCount, -1); count == 0 {
		termbox.Flush()
	}
}

func (ui *Ui) Active() string {
	return ui.activeElement.Name
}

func (ui *Ui) SetActive(name string) {
	element, _ := ui.elements[name]
	if ui.activeElement != nil {
		ui.activeElement.View.uiSetActive(false)
	}
	ui.activeElement = element
	if element != nil {
		element.View.uiSetActive(true)
	}
	ui.Refresh()
}

func (ui *Ui) Run() error {

	ui.manager.OnUiInitialize()

	for {
		select {
		case <-ui.close:
			return nil
		}
	}
}

func (ui *Ui) onCharacterEvent(ch rune) {
	if ui.activeElement != nil {
		ui.activeElement.View.uiCharacterEvent(ch)
	}
}

func (ui *Ui) onKeyEvent(mod Modifier, key Key) {
	if ui.keyListeners[key] != nil {
		for _, listener := range ui.keyListeners[key] {
			listener(ui, key)
		}
	}
	if ui.activeElement != nil {
		ui.activeElement.View.uiKeyEvent(mod, key)
	}
}

func (ui *Ui) Add(name string, view View) error {
	if _, ok := ui.elements[name]; ok {
		return errors.New("view already exists")
	}
	ui.elements[name] = &uiElement{
		Name: name,
		View: view,
	}
	view.uiInitialize(ui)
	return nil
}

func (ui *Ui) SetBounds(name string, x0, y0, x1, y1 int) error {
	element, ok := ui.elements[name]
	if !ok {
		return errors.New("view does not exist")
	}
	element.X0, element.Y0, element.X1, element.Y1 = x0, y0, x1, y1
	element.View.uiSetBounds(x0, y0, x1, y1)
	return nil
}

func (ui *Ui) AddKeyListener(listener KeyListener, key Key) {
	ui.keyListeners[key] = append(ui.keyListeners[key], listener)
}
