package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	"github.com/anthonynsimon/bild/effect"
	"github.com/anthonynsimon/bild/imgio"

	// Audio
	"github.com/faiface/beep"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
)

// ===== AUDIO =====

var musicCtrl *beep.Ctrl
var musicPlaying bool

func PlayMusicLoop(path string) error {
	if musicPlaying {
		return nil
	}

	f, err := os.Open(path)
	if err != nil {
		return err
	}

	streamer, format, err := mp3.Decode(f)
	if err != nil {
		return err
	}

	loop := beep.Loop(-1, streamer)

	speaker.Init(format.SampleRate, format.SampleRate.N(time.Second/10))
	musicCtrl = &beep.Ctrl{Streamer: loop, Paused: false}
	speaker.Play(musicCtrl)

	musicPlaying = true
	return nil
}

func PlaySoundEffect(path string) {
	f, err := os.Open(path)
	if err != nil {
		fmt.Println("SFX error:", err)
		return
	}

	streamer, _, err := mp3.Decode(f)
	if err != nil {
		fmt.Println("SFX decode error:", err)
		return
	}

	speaker.Play(beep.Seq(streamer, beep.Callback(func() {
		_ = f.Close()
	})))
}

// ===== UNIVERSAL CLICK INTERCEPTOR =====

type ClickInterceptor struct {
	widget.BaseWidget
	content fyne.CanvasObject
}

func NewClickInterceptor(content fyne.CanvasObject) *ClickInterceptor {
	c := &ClickInterceptor{content: content}
	c.ExtendBaseWidget(c)
	return c
}

func (c *ClickInterceptor) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(c.content)
}

func (c *ClickInterceptor) MouseDown(*fyne.PointEvent) {
	PlaySoundEffect("sfx/click.mp3")
}

// ===== GAME DATA =====

type Animal struct {
	Name          string   `json:"Name"`
	Level         int      `json:"Level"`
	Mobility      string   `json:"Mobility"`
	Intelligence  int      `json:"Intelligence"`
	Contacts      []string `json:"Contacts"`
	Infected      bool     `json:"Infected"`
	InfectionRate float64  `json:"InfectionRate"`
	Location      string   `json:"Location"`
	RedHerring    bool     `json:"RedHerring"`
}

func (a *Animal) GetImagePath() string {
	return fmt.Sprintf("png/%s.png", a.Name)
}

type Virus struct {
	Modes    []string
	Strength float64
}

type RedHerringInfo struct {
	FunFact string `json:"FunFact"`
	Reason  string `json:"Reason"`
}

type Stats struct {
	Attempts            int
	SameLevelInfections int
	NextLevelInfections int
	StartTime           time.Time
}

type GameState struct {
	animals    map[string]*Animal
	playerName string
	maxLevel   int
	currentDay int
	virus      *Virus
	stats      Stats
	timerStop  chan bool
	redFacts   map[string]RedHerringInfo
	score      int
}

// ===== LOADING =====

func LoadAnimalsFromJSON(path string) (map[string]*Animal, int) {
	data, _ := ioutil.ReadFile(path)

	var raw map[string][]*Animal
	json.Unmarshal(data, &raw)

	result := map[string]*Animal{}
	max := 0
	for _, arr := range raw {
		for _, a := range arr {
			result[a.Name] = a
			if a.Level > max {
				max = a.Level
			}
		}
	}
	return result, max
}

func LoadRedHerringFacts(path string) map[string]RedHerringInfo {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return map[string]RedHerringInfo{}
	}
	var out map[string]RedHerringInfo
	_ = json.Unmarshal(data, &out)
	return out
}

// ===== UI HELPERS =====

func loadBackground() *canvas.Image {
	bg := canvas.NewImageFromFile("yellowstone.png")
	bg.FillMode = canvas.ImageFillStretch
	return bg
}

func loadAnimalImage(path string, invert bool, size float32) *canvas.Image {
	img, err := imgio.Open(path)
	if err != nil {
		return canvas.NewImageFromImage(nil)
	}
	if invert {
		img = effect.Invert(img)
	}
	i := canvas.NewImageFromImage(img)
	i.SetMinSize(fyne.NewSize(size, size))
	i.FillMode = canvas.ImageFillContain
	return i
}

// ===== SCORE =====

func calculateScore(state *GameState) int {
	secs := int(time.Since(state.stats.StartTime).Seconds())
	score := 1000 + (state.stats.NextLevelInfections * 200) - (state.stats.SameLevelInfections * 100) - (state.stats.Attempts * 10) - secs/2
	if score < 0 {
		score = 0
	}
	return score
}

// ===== ANIMATION =====

func showSpookyAnimation(win fyne.Window, state *GameState, imgPath, name string, after func()) {
	bg := loadBackground()
	img := loadAnimalImage(imgPath, true, 430)

	txt := canvas.NewText(fmt.Sprintf("…%s has fallen…", name), color.White)
	txt.TextSize = 34
	txt.Alignment = fyne.TextAlignCenter

	body := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(txt),
		container.NewCenter(img),
		layout.NewSpacer(),
	)

	win.SetContent(NewClickInterceptor(container.NewMax(bg, container.NewCenter(body))))

	go func() {
		for _, size := range []float32{430, 520, 460, 560, 430} {
			time.Sleep(300 * time.Millisecond)
			img.SetMinSize(fyne.NewSize(size, size))
			img.Refresh()
		}
		time.Sleep(600 * time.Millisecond)
		after()
	}()
}

// ===== SCREENS =====

func createWinScreen(app fyne.App, win fyne.Window, state *GameState) fyne.CanvasObject {

	// Play victory sound once
	go func() {
		// Small delay so UI loads first (prevents the thread warning)
		time.Sleep(200 * time.Millisecond)
		fyne.Do(func() {
			PlaySoundEffect("sfx/victory.mp3")
		})

	}()

	finalScore := calculateScore(state)

	title := canvas.NewText("👑 APEX PREDATOR REACHED 👑", color.White)
	title.TextSize = 40
	title.Alignment = fyne.TextAlignCenter

	cleanName := strings.TrimSpace(strings.ToValidUTF8(state.playerName, ""))

	info := canvas.NewText(fmt.Sprintf("Final Host: %s — Score: %d", cleanName, finalScore), color.White)
	info.TextSize = 28
	info.Alignment = fyne.TextAlignCenter

	return NewClickInterceptor(container.NewMax(
		loadBackground(),
		container.NewCenter(
			container.NewVBox(
				layout.NewSpacer(),
				title,
				info,
				layout.NewSpacer(),
			),
		),
	))
}

func createGameScreen(app fyne.App, win fyne.Window, state *GameState) fyne.CanvasObject {

	if state.timerStop != nil {
		state.timerStop <- true
	}
	state.timerStop = make(chan bool)

	timerText := canvas.NewText("⏱ 0s", color.White)
	scoreText := canvas.NewText(fmt.Sprintf("Score: %d", calculateScore(state)), color.White)

	go func() {
		for {
			select {
			case <-state.timerStop:
				return
			default:
				time.Sleep(1 * time.Second)
				timerText.Text = fmt.Sprintf("⏱ %ds", int(time.Since(state.stats.StartTime).Seconds()))
				scoreText.Text = fmt.Sprintf("Score: %d", calculateScore(state))
				timerText.Refresh()
				scoreText.Refresh()
			}
		}
	}()

	player := state.animals[state.playerName]

	header := container.NewVBox(
		container.NewCenter(widget.NewLabelWithStyle(fmt.Sprintf("Day %d — %s (Level %d)", state.currentDay, player.Name, player.Level), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
		container.NewCenter(timerText),
		container.NewCenter(scoreText),
	)

	var cards []fyne.CanvasObject

	for _, target := range state.animals {

		if target.Infected || (target.Level != player.Level && target.Level != player.Level+1) {
			continue
		}

		img := loadAnimalImage(target.GetImagePath(), false, 160)
		name := widget.NewLabel(target.Name)

		btn := widget.NewButton("INFECT", func(t *Animal) func() {
			return func() {

				state.stats.Attempts++

				if t.RedHerring {
					PlaySoundEffect("sfx/fail.mp3")
					info := state.redFacts[t.Name]
					dialog.ShowInformation("🚫 RED HERRING", fmt.Sprintf("%s cannot be infected.\n🐾 %s\n📌 %s", t.Name, info.FunFact, info.Reason), win)
					return
				}

				if rand.Float64() < t.InfectionRate*state.virus.Strength {
					PlaySoundEffect("sfx/success.mp3")
					t.Infected = true
					state.currentDay++

					if t.Level > player.Level {
						state.stats.NextLevelInfections++
					} else {
						state.stats.SameLevelInfections++
					}

					showSpookyAnimation(win, state, t.GetImagePath(), t.Name, func() {

						state.playerName = t.Name

						if t.Level == state.maxLevel {
							win.SetContent(createWinScreen(app, win, state))
							return
						}

						win.SetContent(createGameScreen(app, win, state))
					})

					return
				}

				PlaySoundEffect("sfx/fail.mp3")
				dialog.ShowInformation("Failed", t.Name+" resisted infection.", win)
			}
		}(target))

		card := container.NewVBox(container.NewCenter(img), container.NewCenter(name), container.NewCenter(btn))
		cards = append(cards, card)
	}

	grid := container.NewGridWithColumns(3, cards...)

	return NewClickInterceptor(container.NewMax(loadBackground(),
		container.NewBorder(header, nil, nil, nil, container.NewScroll(grid))))
}

func createStarterSelectionScreen(app fyne.App, win fyne.Window, state *GameState) fyne.CanvasObject {
	var cards []fyne.CanvasObject

	for _, a := range state.animals {
		if a.Level != 1 {
			continue
		}

		img := loadAnimalImage(a.GetImagePath(), false, 160)
		name := widget.NewLabel(a.Name)

		btn := widget.NewButton("Choose", func(an *Animal) func() {
			return func() {

				if an.RedHerring {
					PlaySoundEffect("sfx/fail.mp3")
					info := state.redFacts[an.Name]
					dialog.ShowInformation("🚫 Cannot Start Here",
						fmt.Sprintf("%s cannot be patient zero.\n🐾 %s\n📌 %s", an.Name, info.FunFact, info.Reason), win)
					return
				}

				PlaySoundEffect("sfx/success.mp3")

				state.playerName = an.Name
				an.Infected = true
				state.stats.StartTime = time.Now()

				win.SetContent(createGameScreen(app, win, state))
			}
		}(a))

		card := container.NewVBox(container.NewCenter(img), container.NewCenter(name), container.NewCenter(btn))
		cards = append(cards, card)
	}

	return NewClickInterceptor(container.NewMax(loadBackground(),
		container.NewBorder(widget.NewLabelWithStyle("Choose Your Patient Zero", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
			nil, nil, nil, container.NewScroll(container.NewGridWithColumns(3, cards...)))))
}

func createIntroScreen(app fyne.App, win fyne.Window, state *GameState) fyne.CanvasObject {

	title := widget.NewLabelWithStyle("🦠 YELLOWSTONE OUTBREAK 🦠",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	sub := widget.NewLabel(
		"Deep beneath the soil of Yellowstone National Park, something ancient awakens.\n" +
			"For millennia, it has slept. Dormant, waiting.\n\n" +
			"But now, conditions are perfect.\n" +
			"A new pathogen emerges — adaptable, hungry, unstoppable.\n\n" +
			"Your mission:\n" +
			"• Start at the bottom of the food chain.\n" +
			"• Infect wildlife.\n" +
			"• Evolve.\n\n" +
			"⚠️ Some species are RED HERRINGS — a dead end.\n" +
			"Choose wisely.\n\n" +
			"Will you rise to become Yellowstone’s apex pathogen?",
	)

	sub.Alignment = fyne.TextAlignCenter

	start := widget.NewButton("Begin Infection", func() {
		win.SetContent(createStarterSelectionScreen(app, win, state))
	})

	return NewClickInterceptor(container.NewMax(
		loadBackground(),
		container.NewCenter(container.NewVBox(layout.NewSpacer(), title, sub, layout.NewSpacer(), start, layout.NewSpacer())),
	))
}

// ===== MAIN =====

func main() {
	rand.Seed(time.Now().UnixNano())

	application := app.New()
	win := application.NewWindow("🦠 Yellowstone Outbreak")
	win.Resize(fyne.NewSize(1200, 800))

	animals, max := LoadAnimalsFromJSON("yellowstone_animals.json")

	state := &GameState{
		animals:  animals,
		maxLevel: max,
		virus: &Virus{
			Modes:    []string{"Bite"},
			Strength: 1.0,
		},
		redFacts: LoadRedHerringFacts("red_herring_facts.json"),
		stats:    Stats{StartTime: time.Now()},
	}

	_ = PlayMusicLoop("music/background.mp3")

	win.SetContent(createIntroScreen(application, win, state))
	win.ShowAndRun()
}
