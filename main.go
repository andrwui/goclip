package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type ClipItem struct {
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

type ClipboardManager struct {
	history     []ClipItem
	storagePath string
	window      fyne.Window
	list        *widget.List
	lastContent string
}

func NewClipboardManager() *ClipboardManager {
	home, _ := os.UserHomeDir()
	return &ClipboardManager{
		history:     make([]ClipItem, 0),
		storagePath: filepath.Join(home, ".local", "share", "goclip-history.json"),
	}
}

func (cm *ClipboardManager) getClipboardContent() (string, error) {
	cmd := exec.Command("wl-paste")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func (cm *ClipboardManager) addToHistory(content string) {
	if content == "" {
		return
	}

	// Check for existing content and remove it
	for i, item := range cm.history {
		if item.Content == content {
			cm.history = append(cm.history[:i], cm.history[i+1:]...)
			break
		}
	}

	// Add to top
	cm.history = append([]ClipItem{{
		Content:   content,
		Timestamp: time.Now(),
	}}, cm.history...)

	if len(cm.history) > 100 {
		cm.history = cm.history[:100]
	}

	go cm.saveHistory()
	if cm.list != nil {
		cm.list.Refresh()
	}
}

func (cm *ClipboardManager) createUI() {
	a := app.New()
	a.Settings().SetTheme(theme.DarkTheme())

	cm.window = a.NewWindow("goclip")
	cm.window.SetFixedSize(true)

	cm.list = widget.NewList(
		func() int { return len(cm.history) },
		func() fyne.CanvasObject {
			return container.NewHBox(
				widget.NewLabel(strings.Repeat(" ", 50)),
				widget.NewButton("Copy", nil),
				widget.NewButton("Delete", nil),
			)
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			box := o.(*fyne.Container)
			label := box.Objects[0].(*widget.Label)
			copyBtn := box.Objects[1].(*widget.Button)
			deleteBtn := box.Objects[2].(*widget.Button)

			content := cm.history[i].Content
			if len(content) > 50 {
				content = content[:47] + "..."
			}
			content = strings.ReplaceAll(content, "\n", " ")
			label.SetText(content)

			copyBtn.OnTapped = func() {
				cmd := exec.Command("wl-copy", cm.history[i].Content)
				cmd.Run()
			}

			deleteBtn.OnTapped = func() {
				if len(cm.history) > 1 {
					cm.history = append(cm.history[:i], cm.history[i+1:]...)
					go cm.saveHistory()
					cm.list.Refresh()
				} else if len(cm.history) == 1 {
					cm.history = []ClipItem{}
					go cm.saveHistory()
					cm.list.Refresh()
				}
			}
		},
	)

	content := container.NewBorder(
		nil, nil, nil, nil,
		cm.list,
	)

	cm.window.Resize(fyne.NewSize(600, 400))
	cm.window.SetContent(content)
	cm.window.CenterOnScreen()
}

func (cm *ClipboardManager) monitor() {
	ticker := time.NewTicker(time.Second)
	for range ticker.C {
		if content, err := cm.getClipboardContent(); err == nil {
			cm.addToHistory(content)
		}
	}
}

func main() {
	cm := NewClipboardManager()
	if data, err := os.ReadFile(cm.storagePath); err == nil {
		json.Unmarshal(data, &cm.history)
	}

	cm.createUI()
	go cm.monitor()
	cm.window.ShowAndRun()
}

func (cm *ClipboardManager) saveHistory() error {
	data, err := json.Marshal(cm.history)
	if err != nil {
		return err
	}
	return os.WriteFile(cm.storagePath, data, 0644)
}

