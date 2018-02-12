package editor

import (
	"fmt"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"
)

// Frame is
type Frame struct {
	vertical bool
	width    int
	height   int
	x        int
	y        int
	editor   *Editor
	children []*Frame
	parent   *Frame
	win      *Window
}

func (f *Frame) split(vertical bool) {
	if f.hasChildren() {
		fmt.Println("split has children already")
		return
	}
	win := f.win
	if win == nil {
		return
	}
	newFrame := &Frame{editor: f.editor}
	newWin := NewWindow(win.editor, newFrame)
	newWin.loadBuffer(win.buffer)

	parent := f.parent
	if parent != nil && parent.vertical == vertical {
		newFrame.parent = parent
		children := []*Frame{}
		for _, child := range parent.children {
			if child == f {
				children = append(children, child)
				children = append(children, newFrame)
			} else {
				children = append(children, child)
			}
		}
		parent.children = children
	} else {
		newFrame.parent = f
		frame := &Frame{
			parent: f,
			win:    win,
			editor: f.editor,
		}
		win.frame = frame
		f.children = []*Frame{}
		f.vertical = vertical
		f.win = nil
		f.children = append(f.children, frame, newFrame)
	}
	win.editor.equalWins()
	win.view.SetFocus2()
}

func (f *Frame) hasChildren() bool {
	return f.children != nil && len(f.children) > 0
}

func (f *Frame) setPos(x, y int) {
	f.x = x
	f.y = y
	if !f.hasChildren() {
		fmt.Println("set x y", x, y)
		return
	}

	for _, child := range f.children {
		child.setPos(x, y)
		if f.vertical {
			x += child.width
		} else {
			y += child.height
		}
	}
}

func (f *Frame) setSize(vertical bool, singleValue int) {
	if !f.hasChildren() {
		if vertical {
			f.width = singleValue
		} else {
			f.height = singleValue
		}
		return
	}

	max := f.countSplits(vertical)
	if vertical {
		f.width = max * singleValue
	} else {
		f.height = max * singleValue
	}

	if f.vertical == vertical {
		for _, child := range f.children {
			child.setSize(vertical, singleValue)
		}
		return
	}

	for _, child := range f.children {
		n := child.countSplits(vertical)
		child.setSize(vertical, singleValue*max/n)
	}
}

func (f *Frame) exchange() {
	parent := f.parent
	if parent == nil {
		return
	}
	if len(parent.children) == 1 {
		parent.exchange()
		return
	}
	i := 0
	for index, child := range parent.children {
		if child == f {
			i = index
			break
		}
	}

	if i == len(parent.children)-1 {
		parent.children[i], parent.children[i-1] = parent.children[i-1], parent.children[i]
	} else {
		parent.children[i], parent.children[i+1] = parent.children[i+1], parent.children[i]
	}
	f.editor.equalWins()
	parent.children[i].setFocus()
}

func (f *Frame) setFocus() {
	if f.hasChildren() {
		f.children[0].setFocus()
		return
	}
	f.win.view.SetFocus2()
}

func (f *Frame) close() *Frame {
	if f.hasChildren() {
		return nil
	}
	if f.parent == nil {
		return nil
	}
	parent := f.parent
	children := []*Frame{}
	i := 0
	for index, child := range parent.children {
		if child != f {
			children = append(children, child)
		} else {
			i = index
		}
	}
	var newFocus *Frame
	parent.children = children
	if len(children) == 0 {
		newFocus = parent.close()
	} else {
		if i > 0 {
			i--
		}
		newFocus = children[i]
	}
	win := f.win
	if win == nil {
		return newFocus
	}
	editor := win.editor
	editor.winsRWMutext.Lock()
	delete(editor.wins, win.id)
	editor.winsRWMutext.Unlock()
	win.view.Hide()
	editor.equalWins()
	if newFocus != nil {
		newFocus.setFocus()
	}
	return newFocus
}

func (f *Frame) countSplits(vertical bool) int {
	if !f.hasChildren() {
		return 1
	}
	n := 0
	if f.vertical == vertical {
		for _, child := range f.children {
			n += child.countSplits(vertical)
		}
	} else {
		for _, child := range f.children {
			v := child.countSplits(vertical)
			if v > n {
				n = v
			}
		}
	}
	return n
}

// Window is for displaying a buffer
type Window struct {
	id     int
	editor *Editor
	view   *widgets.QGraphicsView
	frame  *Frame
	buffer *Buffer
}

// NewWindow creates a new window
func NewWindow(editor *Editor, frame *Frame) *Window {
	editor.winsRWMutext.Lock()
	w := &Window{
		id:     editor.winIndex,
		editor: editor,
		frame:  frame,
		view:   widgets.NewQGraphicsView(nil),
	}
	frame.win = w
	editor.winIndex++
	editor.wins[w.id] = w
	editor.winsRWMutext.Unlock()

	w.view.ConnectKeyPressEvent(func(event *gui.QKeyEvent) {
		if w.buffer == nil {
			return
		}
		if event.Modifiers()&core.Qt__ControlModifier > 0 {
			switch string(event.Key()) {
			case "V":
				fmt.Println("split vertical")
				w.frame.split(true)
				return
			case "S":
				fmt.Println("split horizontal")
				w.frame.split(false)
				return
			case "W":
				fmt.Println("close split")
				w.frame.close()
				return
			case "X":
				w.frame.exchange()
				return
			}
			return
		}

		switch core.Qt__Key(event.Key()) {
		case core.Qt__Key_Return, core.Qt__Key_Enter:
			w.buffer.xiView.InsertNewline()
			return
		case core.Qt__Key_Up:
			w.buffer.xiView.MoveUp()
			return
		case core.Qt__Key_Down:
			w.buffer.xiView.MoveDown()
			return
		case core.Qt__Key_Right:
			w.buffer.xiView.MoveRight()
			return
		case core.Qt__Key_Left:
			w.buffer.xiView.MoveLeft()
			return
		case core.Qt__Key_Tab, core.Qt__Key_Backtab:
			w.buffer.xiView.InsertTab()
			return
		case core.Qt__Key_Backspace:
			w.buffer.xiView.DeleteBackward()
			return
		case core.Qt__Key_Delete:
			w.buffer.xiView.DeleteForward()
			return
		case core.Qt__Key_Escape:
			return
		default:
		}
		w.buffer.xiView.Insert(event.Text())
	})
	w.view.ConnectScrollContentsBy(func(dx, dy int) {
		w.view.ScrollContentsByDefault(dx, dy)
	})
	w.view.SetAlignment(core.Qt__AlignLeft | core.Qt__AlignTop)
	w.view.SetParent(editor.centralWidget)

	return w
}

func (w *Window) loadBuffer(buffer *Buffer) {
	w.buffer = buffer
	w.view.SetScene(buffer.scence)
}
