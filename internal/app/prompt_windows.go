package app

import (
	"errors"
	"runtime"
	"strings"

	"github.com/lxn/walk"
	ui "github.com/lxn/walk/declarative"
)

func platformPrompt(req promptRequest) (promptResult, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	var dialog *walk.Dialog
	var submit, cancel *walk.PushButton
	fields := make([]*walk.LineEdit, len(req.Names))
	message := req.Message
	if strings.TrimSpace(message) == "" {
		message = "Paste the requested secrets."
	}
	caption := func(s string) ui.TextLabel {
		return ui.TextLabel{Text: s, NoPrefix: true, MinSize: ui.Size{Width: 100}}
	}
	rows := []ui.Widget{caption(message)}
	for i, name := range req.Names {
		rows = append(rows, caption(name), ui.LineEdit{AssignTo: &fields[i], PasswordMode: true, MaxLength: 0x7ffffffe, Accessibility: ui.Accessibility{Name: name}})
	}
	rows = append(rows, ui.VSpacer{})
	var result promptResult
	accept := func() {
		for _, field := range fields {
			if field.Text() == "" {
				field.SetCueBanner("Required")
				field.SetFocus()
				return
			}
		}
		result.Values = make(map[string]string, len(fields))
		for i, field := range fields {
			result.Values[req.Names[i]] = field.Text()
		}
		dialog.Accept()
	}
	icon, err := walk.NewIconFromResourceId(2)
	if err != nil {
		return promptResult{}, errors.New("could not load the Fidelius dialog resources")
	}
	defer icon.Dispose()
	form := ui.Dialog{
		AssignTo: &dialog, Title: "Fidelius", Icon: icon,
		DefaultButton: &submit, CancelButton: &cancel,
		MinSize: ui.Size{Width: 300, Height: 190}, Size: ui.Size{Width: 480, Height: 190 + 64*min(len(fields), 4)},
		Layout: ui.VBox{Margins: ui.Margins{Left: 16, Top: 16, Right: 16, Bottom: 14}, Spacing: 10},
		Children: []ui.Widget{
			ui.ScrollView{HorizontalFixed: true, Layout: ui.VBox{MarginsZero: true, Spacing: 8}, Children: rows},
			caption("Auto-delete in " + req.AutoDeleteLabel + "."),
			ui.Composite{Layout: ui.HBox{MarginsZero: true}, Children: []ui.Widget{
				ui.HSpacer{},
				ui.PushButton{AssignTo: &submit, Text: "Submit", OnClicked: accept},
				ui.PushButton{AssignTo: &cancel, Text: "Cancel", OnClicked: func() { dialog.Cancel() }},
			}},
		},
	}
	if err := form.Create(nil); err != nil {
		return promptResult{}, errors.New("could not create the Fidelius desktop dialog")
	}
	defer dialog.Dispose()
	dialog.Starting().Attach(func() { fields[0].SetFocus() })
	if dialog.Run() != walk.DlgCmdOK {
		return promptResult{Cancelled: true}, nil
	}
	return result, nil
}
