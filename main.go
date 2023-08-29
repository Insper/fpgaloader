package main

import (
	"errors"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/fsnotify/fsnotify"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var APPVERSION string = "0.1.0"
var PROGRAMMING_QUEUE chan string
var TIMESTAMP_LAST_JOB time.Time

func OpenFpgaLoaderVersion() (string, error) {
	Cmd := exec.Command("openFPGALoader", "-V")
	lsOut, err := Cmd.Output()
	if err != nil {
		return "", err
	}

	if !strings.Contains(string(lsOut), "openFPGALoader v") {
		return "", errors.New("Falha ao executar openFPGALoader. " + string(lsOut))
	}
	return string(lsOut), nil
}

func OpenFpgaLoaderDetectBoard() (string, error) {
	Cmd := exec.Command("openFPGALoader", "--scan-usb")
	lsOut, err := Cmd.Output()
	if err != nil {
		return "", err
	}

	if strings.Contains(string(lsOut), "found 0 USB device") {
		return "", errors.New("Não encontrado nenhum dispositivo USB " + string(lsOut))
	}
	return string(lsOut), nil
}

func OpenFpgaLoaderProgramDe0(fileProgram string) (string, error) {
	Cmd := exec.Command("openFPGALoader", "-b", "de0", fileProgram)
	lsOut, err := Cmd.Output()
	if err != nil {
		return "", err
	}

	if !strings.Contains(string(lsOut), "Done") {
		return "", errors.New("Erro ao programar " + string(lsOut))
	}
	return string(lsOut), nil
}

func watchFileForChanges(watcher *fsnotify.Watcher) {
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			PROGRAMMING_QUEUE <- event.Name
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Println("error:", err)
		}
	}
}

func main() {
	PROGRAMMING_QUEUE = make(chan string)
	TIMESTAMP_LAST_JOB = time.Now()
	logContents := "FPGALoader iniciado."
	statusBarContents := "Procurando openFPGALoader..."

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal(err)
	}
	defer watcher.Close()

	go watchFileForChanges(watcher)

	a := app.New()
	w := a.NewWindow("FPGALoader " + APPVERSION)
	w.Resize(fyne.Size{Width: 600, Height: 600})

	edit1 := widget.NewMultiLineEntry()
	edit1.Wrapping = fyne.TextWrapWord
	edit1.MultiLine = true
	edit1.SetText(logContents)
	edit1.OnChanged = func(string) {
		edit1.SetText(logContents)
	}

	button := widget.NewButton("Procurar arquivo", func() {
		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			mfile := reader.URI().Path()
			absMediaFile, _ := filepath.Abs(mfile)
			watcher.Add(absMediaFile)
			PROGRAMMING_QUEUE <- absMediaFile
		}, w)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".rbf"}))
		fd.Show()
	})

	status := widget.NewEntry()
	status.MultiLine = false
	status.SetText(statusBarContents)
	status.OnChanged = func(string) {
		status.SetText(statusBarContents)
	}

	content := container.NewBorder(button, status, nil, nil, edit1)
	w.SetContent(content)

	ver, err := OpenFpgaLoaderVersion()
	if err != nil {
		dialog.ShowError(err, w)
	} else {
		statusBarContents = ver
		status.SetText(statusBarContents)
	}

	w.SetOnDropped(func(position fyne.Position, uris []fyne.URI) {
		if strings.Contains(uris[0].Path(), ".rbf") {
			watcher.Add(uris[0].Path())
			PROGRAMMING_QUEUE <- uris[0].Path()
		} else {
			logContents += "\nArquivo com extensão inválida"
			edit1.SetText(logContents)
		}
	})

	ticker := time.NewTicker(1000 * time.Millisecond)
	done := make(chan bool)
	go func() {
		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				_, err := OpenFpgaLoaderDetectBoard()
				if err == nil {
					logContents += "\nPlaca encontrada"
					edit1.SetText(logContents)
					done <- true
				}
			}
		}
	}()

	go func() {
		for {
			select {
			case file := <-PROGRAMMING_QUEUE:
				if time.Now().Sub(TIMESTAMP_LAST_JOB) > 1*time.Second {
					logContents += "\nProgramando arquivo " + file
					edit1.SetText(logContents)
					output, err := OpenFpgaLoaderProgramDe0(file)
					if err != nil {
						dialog.ShowError(err, w)
						logContents += "\nErro: " + output
					} else {
						logContents += "\nPlaca programada"
					}
					edit1.SetText(logContents)
					TIMESTAMP_LAST_JOB = time.Now()
				}
			}
		}
	}()

	w.ShowAndRun()

}
