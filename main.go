package main

import (
	"fmt"
	"log"
	"time"
	"os/user"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/dialog"

	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/cristalhq/otp"
)


func main() {
	myApp := app.NewWithID("totpgen")
	myWindow := myApp.NewWindow("TOTP Generator")
	myWindow.SetMaster()

	service := "totpgen"
	curUser, err := user.Current()
	if err != nil {
		log.Fatalf("Unable to get current user: %v", err)
	}

	totp, err := otp.NewTOTP(otp.TOTPConfig {
		Algo: otp.AlgorithmSHA1,
		Digits: 6,
		Issuer: "sf",
		Period: 30,
		Skew: 2,
	})
	if err != nil {
		log.Println(err)
	}

	data := binding.BindStringList(
		&[]string{},
	)
	totps := getSites(service, curUser.Username)
	for k, v := range totps {
		addRecord(totps, service, curUser.Username, k, v, data)
	}

	// Display bar for current TOTP code
	totpCode := canvas.NewText("", nil)
	totpCode.Alignment = fyne.TextAlignCenter
	totpCode.TextSize = 36.0
	totpCode.TextStyle.Monospace = true

	// Progres bar that displays with current selected site
	progress := widget.NewProgressBar()
	progress.Min = 0
	progress.Max = 100
	progress.TextFormatter = func() string { return "" }
	progress.Hide()

	// Form element for Site being added
	siteName := widget.NewEntry()
	siteName.SetPlaceHolder("Site Name")
	siteName.Validator = func(s string) (err error) {
		_, ok := totps[s]
		if ok {
			return fmt.Errorf("site already exists: %s", s)
		}
		return nil
	}
	// Form element for site's secret
	totpSecret := widget.NewEntry()
	totpSecret.SetPlaceHolder("Enter valid key")
	totpSecret.Validator = func(s string) (err error) {
		at := time.Now()
		_, err = totp.GenerateCode(s, at)
		if err != nil {
			return err
		}
		return nil
	}
	
	// Combine elements of new site form
	items := []*widget.FormItem{
		widget.NewFormItem("SiteName", siteName),
		widget.NewFormItem("Secret", totpSecret),
	}
	
	// by default no item is selected
	currentList := -1

	// List to hold list of sites
	list := widget.NewListWithData(data,
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i binding.DataItem, o fyne.CanvasObject) {
			o.(*widget.Label).Bind(i.(binding.String))
		})
	list.OnSelected = func(id widget.ListItemID) {
		currentList = id
		progress.Show()
		progress.Refresh()
	}
	list.OnUnselected = func(id widget.ListItemID) {
		currentList = -1
	}

	// Form Dialog to add new sites
	d := dialog.NewForm("Add Entry", "OK", "Cancel", items, func(b bool) {
		if b {
			err := addRecord(totps, service, curUser.Username, siteName.Text, totpSecret.Text, data)
			if err != nil {
				log.Println(err)
			}
			siteName.Text = ""
			totpSecret.Text = ""
			list.UnselectAll()
			list.Select(list.Length()-1)
			return
		}
	}, myWindow)
	
	// Toolbar at bottom of main screen
	toolbar := widget.NewToolbar(
		widget.NewToolbarAction(theme.ContentAddIcon(), func() {
			d.Resize(fyne.NewSize(550,300))
			d.Show()
		}),
		widget.NewToolbarSeparator(),
		widget.NewToolbarAction(theme.DeleteIcon(), func() {
			if currentList >= 0 {
				dialog.ShowConfirm("Confirm Delete", "Are you sure?", func(b bool) {
					if b {
						val, err := data.GetValue(currentList)
						if err != nil {
							log.Printf("Failed to get current selection: %v\n", err)
						}
						delete(totps, val)
						data.Remove(val)
						currentList = -1
						list.UnselectAll()
						progress.Hide()
						saveSites(service, curUser.Username, totps)
					} else {
					log.Println("Do NOT DELETE")
					}
				}, myWindow)
			} else {
				log.Println("Nothing selected.")
			}
		}),
		widget.NewToolbarSpacer(),
		widget.NewToolbarAction(theme.HelpIcon(), func() {
			
			log.Println("Display help")
			helpWindow := myApp.NewWindow("TOTP Generator Help")
			helpInfo := widget.NewRichTextFromMarkdown(helpText)
			helpInfo.Wrapping = fyne.TextWrapWord
			helpWindow.SetContent(helpInfo)
			helpWindow.Resize(fyne.NewSize(400,400))
			helpWindow.Show()
			helpWindow.RequestFocus()
			
		}),
	)

	// loops to display a TOTP code if a site is selected
	go func() {
		for {
			if currentList >= 0 {
				curSite, _ := data.GetValue(currentList)
				secretInBase32 := totps[curSite]
				at := time.Now()
				code, err := totp.GenerateCode(secretInBase32, at)
				if err != nil {
					totpCode.Text = "Error"
					log.Printf("totp error: %v", err)
				} else { 
					totpCode.Text = code[:3] + " " + code[3:] 
					totpCode.Refresh()
				}
			} else {
				totpCode.Text = ""
				totpCode.Refresh()
			}
			sec := time.Now().Second()
			ns := time.Now().Nanosecond()
			adjSec := (float64(sec % 30.0) + (float64(ns) / 1000000000.0)) * (100.0 / 30.0)
			progress.SetValue(100.0 - adjSec)
			time.Sleep(time.Second / 10)
		}
	}()
	doublebar := container.NewVBox(totpCode, progress, toolbar)

	content := container.NewBorder(nil, doublebar, nil, nil, list)
	myWindow.SetContent(content)
	myWindow.Resize(fyne.NewSize(600,600))
	myWindow.ShowAndRun()
}

func addRecord(t map[string]string, service, user, siteName, secret string, data binding.ExternalStringList) error {
	t[siteName] = secret
	err := data.Append(siteName)
	if err != nil {
		return fmt.Errorf("failed to add site to display list: %v", err)
	}
	err = data.Reload()
	if err != nil {
		return fmt.Errorf("failed to reload display list: %v", err)
	}
	err = saveSites(service, user, t)
	if err != nil {
		return fmt.Errorf("failed to save to keychain: %v", err)
	}
	return nil
}