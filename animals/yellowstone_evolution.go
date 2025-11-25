package main

import (
	"encoding/json"
	"fmt"
	"image/color"
	"io/ioutil"
	"math/rand"
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
)

// -------- DATA --------

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
	return fmt.Sprintf("animals/%s.png", a.Name)
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

// -------- LOADING --------

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

// -------- UI HELPERS --------

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

// -------- GAME MATH --------

func calculateScore(state *GameState) int {
	secs := int(time.Since(state.stats.StartTime).Seconds())
	score := 1000 + (state.stats.NextLevelInfections * 200) - (state.stats.SameLevelInfections * 100) - (state.stats.Attempts * 10) - secs/2
	if score < 0 {
		score = 0
	}
	return score
}

// -------- ANIMATION --------

func showSpookyAnimation(win fyne.Window, state *GameState, imgPath, name string, after func()) {
	bg := loadBackground()
	img := loadAnimalImage(imgPath, true, 430)

	txt := canvas.NewText(fmt.Sprintf("…%s has fallen…", name), color.White)
	txt.TextSize = 34
	txt.Alignment = fyne.TextAlignCenter

	containerBody := container.NewVBox(
		layout.NewSpacer(),
		container.NewCenter(txt),
		container.NewCenter(img),
		layout.NewSpacer(),
	)

	win.SetContent(container.NewMax(bg, container.NewCenter(containerBody)))

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

// -------- SCREENS --------

func createWinScreen(app fyne.App, win fyne.Window, state *GameState) fyne.CanvasObject {

	finalScore := calculateScore(state)

	// BIG victory title
	title := canvas.NewText("👑 APEX PREDATOR REACHED 👑", color.White)
	title.TextSize = 40
	title.Alignment = fyne.TextAlignCenter

	// sanitize name before printing
	cleanName := strings.TrimSpace(strings.ToValidUTF8(state.playerName, ""))

	// Bigger score text
	info := canvas.NewText(fmt.Sprintf("Final Host: %s Score: %d",
		cleanName, finalScore), color.White)
	info.TextSize = 28
	info.Alignment = fyne.TextAlignCenter

	return container.NewMax(
		loadBackground(),
		container.NewCenter(
			container.NewVBox(
				layout.NewSpacer(),
				container.NewCenter(title),
				container.NewCenter(info),
				layout.NewSpacer(),
			),
		),
	)
}


func createGameScreen(app fyne.App, win fyne.Window, state *GameState) fyne.CanvasObject {

	if state.timerStop != nil {
		state.timerStop <- true
	}
	state.timerStop = make(chan bool)

	timerText := canvas.NewText("⏱ 0s", color.White)
	timerText.TextSize = 22
	timerText.Alignment = fyne.TextAlignCenter

	scoreText := canvas.NewText(fmt.Sprintf("Score: %d", calculateScore(state)), color.White)
	scoreText.TextSize = 22
	scoreText.Alignment = fyne.TextAlignCenter

	go func() {
		for {
			select {
			case <-state.timerStop:
				return
			default:
				time.Sleep(time.Second)
				timerText.Text = fmt.Sprintf("⏱ %ds", int(time.Since(state.stats.StartTime).Seconds()))
				scoreText.Text = fmt.Sprintf("Score: %d", calculateScore(state))
				timerText.Refresh()
				scoreText.Refresh()
			}
		}
	}()

	player := state.animals[state.playerName]

	header := container.NewVBox(
		container.NewCenter(widget.NewLabelWithStyle(fmt.Sprintf("Day %d — %s (Level %d)",
			state.currentDay, player.Name, player.Level), fyne.TextAlignCenter, fyne.TextStyle{Bold: true})),
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
					info := state.redFacts[t.Name]
					dialog.ShowInformation("🚫 RED HERRING", fmt.Sprintf("%s cannot be infected.\n\n🐾 %s\n📌 %s",
						t.Name, info.FunFact, info.Reason), win)
					return
				}

				if rand.Float64() < t.InfectionRate*state.virus.Strength {
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
				dialog.ShowInformation("Failed", t.Name+" resisted infection.", win)
			}
		}(target))

		card := container.NewVBox(
			container.NewCenter(img),
			container.NewCenter(name),
			container.NewCenter(btn),
		)

		cards = append(cards, card)
	}

	grid := container.NewGridWithColumns(3, cards...)
	scroll := container.NewScroll(grid)

	return container.NewMax(loadBackground(), container.NewBorder(header, nil, nil, nil, scroll))
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
					info := state.redFacts[an.Name]
					dialog.ShowInformation("🚫 Cannot Start Here",
						fmt.Sprintf("%s cannot be patient zero.\n\n🐾 %s\n📌 %s", an.Name, info.FunFact, info.Reason), win)
					return
				}

				state.playerName = an.Name
				an.Infected = true
				state.stats.StartTime = time.Now()
				win.SetContent(createGameScreen(app, win, state))
			}
		}(a))

		card := container.NewVBox(container.NewCenter(img), container.NewCenter(name), container.NewCenter(btn))
		cards = append(cards, card)
	}

	grid := container.NewGridWithColumns(3, cards...)
	scroll := container.NewScroll(grid)

	header := widget.NewLabelWithStyle("Choose Your Patient Zero", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	return container.NewMax(loadBackground(), container.NewBorder(header, nil, nil, nil, scroll))
}

func createIntroScreen(app fyne.App, win fyne.Window, state *GameState) fyne.CanvasObject {

	title := widget.NewLabelWithStyle("🦠 YELLOWSTONE OUTBREAK 🦠", fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	sub := widget.NewLabel(
		"Deep beneath the soil of Yellowstone National Park, something ancient awakens.\n" +
		"For millennia, it has slept. Dormant, waiting.\n\n" +
		"But now conditions are perfect.\n" +
		"A new pathogen has emerged. Adaptable, hungry, and capable of evolving through infection.\n\n" +
		"Your goal is simple:\n" +
		"📍 Start as a low-level host.\n" +
		"📍 Infect animals across Yellowstone.\n" +
		"📍 Evolve by successfully infecting higher-level species.\n\n" +
		"⚠️ Beware: some animals are RED HERRINGS.\n" +
		"They cannot spread your plague: choosing them wastes time.\n\n" +
		"The clock is ticking.\n" +
		"The ecosystem will not remain unprepared forever.\n\n" +
		"Can you climb the food chain…\n" +
		"and become the uncontested APEX pathogen?")

	sub.Alignment = fyne.TextAlignCenter
	sub.TextStyle = fyne.TextStyle{Bold: true}   // <-- THIS BOLDENS IT


	start := widget.NewButton("Begin Infection", func() {
		win.SetContent(createStarterSelectionScreen(app, win, state))
	})

	return container.NewMax(loadBackground(),
		container.NewCenter(container.NewVBox(
			layout.NewSpacer(),
			container.NewCenter(title),
			container.NewCenter(sub),
			layout.NewSpacer(),
			container.NewCenter(start),
			layout.NewSpacer(),
		)),
	)
}


// -------- MAIN --------

func main() {
	rand.Seed(time.Now().Unix())

	application := app.New()
	win := application.NewWindow("🦠 Yellowstone Outbreak")
	win.Resize(fyne.NewSize(1200, 800))

	animals, max := LoadAnimalsFromJSON("yellowstone_animals.json")

	state := &GameState{
		animals: animals,
		maxLevel: max,
		virus: &Virus{
			Modes:    []string{"Bite"},
			Strength: 1.0,
		},
		redFacts: LoadRedHerringFacts("red_herring_facts.json"),
		stats:    Stats{StartTime: time.Now()},
	}

	win.SetContent(createIntroScreen(application, win, state))
	win.ShowAndRun()
}
