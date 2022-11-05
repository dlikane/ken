package main

import (
	"fmt"
	"time"

	"github.com/faiface/beep"
	"github.com/faiface/beep/generators"
	"github.com/faiface/beep/speaker"
	"github.com/getlantern/systray"
	"github.com/itchyny/volume-go"
)

const (
	max_volume  = 100
	frequency   = 20000
	sleep_time  = 10 * time.Millisecond
	sample_rate = 48000
	buffer_size = 4800
)

var (
	menuBeeper *systray.MenuItem
	menuQuit   *systray.MenuItem
	menuBeep   *systray.MenuItem
	lastBeeped = time.Time{}
)

func main() {
	systray.Run(onReady, onExit)
}

func onExit() {
}

func onReady() {
	systray.SetTemplateIcon(icon, icon)
	systray.SetTitle("Beeper")
	systray.SetTooltip("Beeper")
	menuBeeper = systray.AddMenuItemCheckbox("Running", "Toggle on/off", true)
	menuBeep = systray.AddMenuItem("Beep", "Tu-tu")
	systray.AddSeparator()
	menuQuit = systray.AddMenuItem("Quit", "Quit the whole app")

	runTicker()

	for {
		select {
		case <-menuBeeper.ClickedCh:
			if menuBeeper.Checked() {
				menuBeeper.Uncheck()
				menuBeeper.SetTitle("Stopped")
			} else {
				menuBeeper.Check()
				err := doBeep()
				if err != nil {
					panic(err)
				}
			}
		case <-menuBeep.ClickedCh:
			doBeep()
			return
		case <-menuQuit.ClickedCh:
			systray.Quit()
			return
		}
	}
}

func runTicker() {
	ticker := time.NewTicker(15 * time.Second)
	quit := make(chan struct{})
	go func() {
		for {
			select {
			case <-ticker.C:
				if menuBeeper != nil && menuBeeper.Checked() {
					tick()
				}
			case <-quit:
				ticker.Stop()
				return
			}
		}
	}()
	tick()
}

func formatDelta(d time.Duration) string {
	d = d.Round(time.Second)
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second
	return fmt.Sprintf("%d:%02d", m, s)
}

func tick() {
	delta := time.Now().Sub(lastBeeped)
	if delta > 10*time.Minute {
		lastBeeped = time.Now()
		doBeep()
		delta = 0
	}
	menuBeeper.SetTitle(formatDelta(10*time.Minute - delta))
}

func doBeep() error {
	vol, err := volume.GetVolume()
	if err != nil {
		return err
	}
	err = volume.SetVolume(max_volume)
	if err != nil {
		return err
	}
	speaker.Init(beep.SampleRate(sample_rate), buffer_size)
	s, err := generators.SinTone(beep.SampleRate(sample_rate), frequency)
	if err != nil {
		return err
	}
	speaker.Play(s)
	time.Sleep(sleep_time)
	speaker.Clear()
	time.Sleep(sleep_time)
	speaker.Play(s)
	time.Sleep(sleep_time)
	speaker.Clear()
	err = volume.SetVolume(vol)
	if err != nil {
		return err
	}
	return nil
}
