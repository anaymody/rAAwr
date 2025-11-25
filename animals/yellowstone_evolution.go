package main

import (
	"encoding/json"
	"fmt"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io/ioutil"
	"log"
	"math/rand"
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
	_ "golang.org/x/image/webp"
)

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

// GetImagePath constructs the image path from animal name
func (a *Animal) GetImagePath() string {
	return fmt.Sprintf("animals/%s.jpg", a.Name)
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
	animals     map[string]*Animal
	playerName  string
	maxLevel    int
	currentDay  int
	virus       *Virus
	stats       Stats
	redFacts    map[string]RedHerringInfo
}

// Load animals from JSON
func LoadAnimalsFromJSON(filepath string) (map[string]*Animal, int) {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Printf("⚠️  Failed to load JSON file: %v", err)
		return nil, 0
	}

	var raw map[string][]*Animal
	err = json.Unmarshal(data, &raw)
	if err != nil {
		log.Printf("⚠️  Failed to parse JSON: %v", err)
		return nil, 0
	}

	result := map[string]*Animal{}
	maxLevel := 0

	for _, group := range raw {
		for _, animal := range group {
			result[animal.Name] = animal
			if animal.Level > maxLevel {
				maxLevel = animal.Level
			}
		}
	}
	return result, maxLevel
}

// Load red herring facts
func LoadRedHerringFacts(filepath string) map[string]RedHerringInfo {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Printf("⚠️  No red herring facts file found.")
		return map[string]RedHerringInfo{}
	}

	var info map[string]RedHerringInfo
	json.Unmarshal(data, &info)
	return info
}

// Load and optionally invert an image
func loadAnimalImage(imagePath string, inverted bool) *canvas.Image {
	if imagePath == "" {
		placeholder := canvas.NewImageFromImage(nil)
		placeholder.FillMode = canvas.ImageFillContain
		return placeholder
	}

	img, err := imgio.Open(imagePath)
	if err != nil {
		log.Printf("⚠️  Failed to load image %s: %v", imagePath, err)
		placeholder := canvas.NewImageFromImage(nil)
		placeholder.FillMode = canvas.ImageFillContain
		return placeholder
	}

	var displayImg image.Image
	if inverted {
		displayImg = effect.Invert(img)
	} else {
		displayImg = img
	}

	canvasImg := canvas.NewImageFromImage(displayImg)
	canvasImg.FillMode = canvas.ImageFillContain
	canvasImg.SetMinSize(fyne.NewSize(150, 150))
	return canvasImg
}

// Get valid infection targets (same level or +1)
func getValidTargets(player *Animal, animals map[string]*Animal) []*Animal {
	valid := []*Animal{}
	for _, a := range animals {
		if a.Infected || a.RedHerring {
			continue
		}
		if a.Level == player.Level || a.Level == player.Level+1 {
			valid = append(valid, a)
		}
	}
	return valid
}

// Attempt infection
func attemptInfection(player *Animal, target *Animal, virus *Virus, stats *Stats) bool {
	rand.Seed(time.Now().UnixNano())
	chance := target.InfectionRate * virus.Strength
	
	stats.Attempts++
	
	if rand.Float64() < chance {
		target.Infected = true
		
		if target.Level == player.Level {
			stats.SameLevelInfections++
		} else if target.Level == player.Level+1 {
			stats.NextLevelInfections++
		}
		
		return true
	}
	return false
}

// Calculate score
func calculateScore(stats Stats, elapsed time.Duration) int {
	seconds := int(elapsed.Seconds())
	score := 1000
	score += stats.NextLevelInfections * 200
	score -= stats.SameLevelInfections * 100
	score -= stats.Attempts * 10
	score -= seconds / 2
	
	if score < 0 {
		score = 0
	}
	return score
}

// Create intro screen
func createIntroScreen(myApp fyne.App, myWindow fyne.Window, gameState *GameState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"🦠 YELLOWSTONE OUTBREAK 🦠",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	story := widget.NewLabel(
		"Deep in the heart of Yellowstone National Park, an ancient pathogen\n" +
			"has awakened from its slumber in the thermal springs.\n\n" +
			"You are a microscopic organism, newly evolved and hungry for hosts.\n" +
			"Your mission: Climb the food chain by infecting animals, evolving with\n" +
			"each successful jump to a higher-level predator.\n\n" +
			"🔬 START AS A LEVEL 1 ORGANISM\n" +
			"⬆️  EVOLVE BY INFECTING HIGHER-LEVEL HOSTS\n" +
			"🎯 REACH THE APEX PREDATOR AT THE TOP\n" +
			"⚠️  BEWARE: Some animals are red herrings - dead ends that waste your time\n\n" +
			"The food chain awaits. Will you reach the apex?",
	)
	story.Wrapping = fyne.TextWrapWord
	story.Alignment = fyne.TextAlignCenter

	startBtn := widget.NewButton("🎮 BEGIN INFECTION", func() {
		selectScreen := createAnimalSelectionScreen(myApp, myWindow, gameState)
		myWindow.SetContent(selectScreen)
	})
	startBtn.Importance = widget.HighImportance

	content := container.NewVBox(
		layout.NewSpacer(),
		title,
		layout.NewSpacer(),
		story,
		layout.NewSpacer(),
		container.NewCenter(startBtn),
		layout.NewSpacer(),
	)

	return content
}

// Create animal selection screen (Level 1 only, no red herrings)
func createAnimalSelectionScreen(myApp fyne.App, myWindow fyne.Window, gameState *GameState) fyne.CanvasObject {
	title := widget.NewLabelWithStyle(
		"Choose Your Patient Zero",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	instruction := widget.NewLabel("Select a Level 1 organism to begin your climb:")
	instruction.Alignment = fyne.TextAlignCenter

	// Get Level 1 animals only
	var levelOneAnimals []*Animal
	for _, animal := range gameState.animals {
		if animal.Level == 1 {
			levelOneAnimals = append(levelOneAnimals, animal)
		}
	}

	var animalCards []fyne.CanvasObject

	for _, animal := range levelOneAnimals {
		animalData := animal

		img := loadAnimalImage(animal.GetImagePath(), false)

		nameLabel := widget.NewLabel(animalData.Name)
		nameLabel.Alignment = fyne.TextAlignCenter
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}

		statsLabel := widget.NewLabel(fmt.Sprintf(
			"Level: %d | Intelligence: %d\nInfection Rate: %.0f%%",
			animalData.Level,
			animalData.Intelligence,
			animalData.InfectionRate*100,
		))
		statsLabel.Alignment = fyne.TextAlignCenter

		var selectBtn *widget.Button
		
		if animalData.RedHerring {
			selectBtn = widget.NewButton("🚫 Red Herring", func() {
				info, ok := gameState.redFacts[animalData.Name]
				var message string
				if ok {
					message = fmt.Sprintf("🚫 RED HERRING\n\n🐾 Fun Fact: %s\n\n📌 Reason: %s\n\nThis animal cannot be used as a starting point!",
						info.FunFact, info.Reason)
				} else {
					message = "This animal is a red herring and cannot be selected!"
				}
				dialog.ShowInformation("Red Herring", message, myWindow)
			})
			selectBtn.Importance = widget.DangerImportance
		} else {
			selectBtn = widget.NewButton("Choose", func() {
				gameState.playerName = animalData.Name
				gameState.animals[animalData.Name].Infected = true
				gameState.stats.StartTime = time.Now()
				gameScreen := createGameScreen(myApp, myWindow, gameState)
				myWindow.SetContent(gameScreen)
			})
			selectBtn.Importance = widget.HighImportance
		}

		card := container.NewVBox(
			img,
			nameLabel,
			statsLabel,
			selectBtn,
		)

		animalCards = append(animalCards, card)
	}

	grid := container.New(layout.NewGridLayout(3), animalCards...)
	scrollable := container.NewScroll(grid)

	content := container.NewBorder(
		container.NewVBox(title, instruction),
		nil,
		nil,
		nil,
		scrollable,
	)

	return content
}

// Create main game screen
func createGameScreen(myApp fyne.App, myWindow fyne.Window, gameState *GameState) fyne.CanvasObject {
	player := gameState.animals[gameState.playerName]

	elapsed := time.Since(gameState.stats.StartTime)
	
	titleLabel := widget.NewLabelWithStyle(
		fmt.Sprintf("DAY %d", gameState.currentDay),
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	playerLabel := widget.NewLabel(fmt.Sprintf("You are: %s ☣️ (Level %d)", gameState.playerName, player.Level))
	playerLabel.Alignment = fyne.TextAlignCenter

	goalLabel := widget.NewLabel(fmt.Sprintf("Goal: Reach Level %d Apex Predator", gameState.maxLevel))
	goalLabel.Alignment = fyne.TextAlignCenter

	statsLabel := widget.NewLabel(fmt.Sprintf(
		"⏱ Time: %.1fs | 🎯 Attempts: %d | ⬆️ Evolutions: %d",
		elapsed.Seconds(),
		gameState.stats.Attempts,
		gameState.stats.NextLevelInfections,
	))
	statsLabel.Alignment = fyne.TextAlignCenter

	statusLabel := widget.NewLabel("Choose an animal to infect (same level or +1 level):")
	statusLabel.Alignment = fyne.TextAlignCenter

	// Get valid targets
	validTargets := getValidTargets(player, gameState.animals)

	var animalCards []fyne.CanvasObject

	// Show all animals but only enable valid targets
	for _, animal := range gameState.animals {
		animalData := animal

		img := loadAnimalImage(animal.GetImagePath(), animal.Infected)

		nameLabel := widget.NewLabel(fmt.Sprintf("%s (L%d)", animalData.Name, animalData.Level))
		nameLabel.Alignment = fyne.TextAlignCenter
		nameLabel.TextStyle = fyne.TextStyle{Bold: true}

		var statusText string
		if animalData.Infected {
			statusText = "☣️ INFECTED"
		} else if animalData.RedHerring {
			statusText = "🚫 Red Herring"
		} else {
			statusText = "😐 Healthy"
		}
		statusTextLabel := widget.NewLabel(statusText)
		statusTextLabel.Alignment = fyne.TextAlignCenter

		// Check if this is a valid target
		isValidTarget := false
		for _, vt := range validTargets {
			if vt.Name == animalData.Name {
				isValidTarget = true
				break
			}
		}

		var actionBtn *widget.Button
		
		if isValidTarget {
			actionBtn = widget.NewButton(fmt.Sprintf("🎯 Infect (%.0f%%)", animalData.InfectionRate*100), func() {
				infectedAnimal := animalData
				success := attemptInfection(player, infectedAnimal, gameState.virus, &gameState.stats)
				
				var message string
				
				if success {
					if infectedAnimal.Level > player.Level {
						// Evolution!
						message = fmt.Sprintf("💥 SUCCESS!\n\n%s is infected!\n\n🔄 EVOLUTION!\nYou now inhabit %s\n⬆️ Level Up: %d → %d",
							infectedAnimal.Name, infectedAnimal.Name, player.Level, infectedAnimal.Level)
						gameState.playerName = infectedAnimal.Name
						
						// Check for win
						if infectedAnimal.Level == gameState.maxLevel {
							time.AfterFunc(time.Millisecond*500, func() {
								endScreen := createWinScreen(myApp, myWindow, gameState)
								myWindow.SetContent(endScreen)
							})
							dialog.ShowInformation("🏆 APEX REACHED!", 
								fmt.Sprintf("YOU WIN!\n\nYou've reached the apex predator: %s!\n\nFinal score will be calculated...", 
									infectedAnimal.Name), myWindow)
							return
						}
					} else {
						message = fmt.Sprintf("💥 SUCCESS!\n\n%s is now infected!\n(Same level - no evolution)", infectedAnimal.Name)
					}
				} else {
					message = fmt.Sprintf("❌ FAILED\n\n%s resisted the infection.\nTry again!", infectedAnimal.Name)
				}

				dialog.ShowInformation("Infection Attempt", message, myWindow)

				gameState.currentDay++
				time.AfterFunc(time.Millisecond*500, func() {
					gameScreen := createGameScreen(myApp, myWindow, gameState)
					myWindow.SetContent(gameScreen)
				})
			})
			actionBtn.Importance = widget.HighImportance
		} else if animalData.Infected {
			actionBtn = widget.NewButton("Already Infected", func() {})
			actionBtn.Disable()
		} else if animalData.RedHerring {
			actionBtn = widget.NewButton("Red Herring", func() {
				info, ok := gameState.redFacts[animalData.Name]
				var msg string
				if ok {
					msg = fmt.Sprintf("🚫 RED HERRING\n\n🐾 %s\n\n📌 %s", info.FunFact, info.Reason)
				} else {
					msg = "This is a red herring and cannot be infected!"
				}
				dialog.ShowInformation("Red Herring", msg, myWindow)
			})
			actionBtn.Importance = widget.DangerImportance
		} else {
			actionBtn = widget.NewButton(fmt.Sprintf("Level %d (Out of Range)", animalData.Level), func() {})
			actionBtn.Disable()
		}

		card := container.NewVBox(
			img,
			nameLabel,
			statusTextLabel,
			actionBtn,
		)

		animalCards = append(animalCards, card)
	}

	grid := container.New(layout.NewGridLayout(3), animalCards...)
	scrollable := container.NewScroll(grid)

	header := container.NewVBox(
		titleLabel,
		playerLabel,
		goalLabel,
		statsLabel,
		statusLabel,
	)

	content := container.NewBorder(
		header,
		nil,
		nil,
		nil,
		scrollable,
	)

	return content
}

// Create win screen
func createWinScreen(myApp fyne.App, myWindow fyne.Window, gameState *GameState) fyne.CanvasObject {
	elapsed := time.Since(gameState.stats.StartTime)
	finalScore := calculateScore(gameState.stats, elapsed)

	player := gameState.animals[gameState.playerName]

	title := widget.NewLabelWithStyle(
		"🏆 APEX PREDATOR REACHED! 🏆",
		fyne.TextAlignCenter,
		fyne.TextStyle{Bold: true},
	)

	winText := fmt.Sprintf(
		"You successfully evolved to: %s\n\nLevel: %d (APEX PREDATOR)",
		player.Name, player.Level,
	)
	winLabel := widget.NewLabel(winText)
	winLabel.Alignment = fyne.TextAlignCenter

	scoreText := fmt.Sprintf(
		"⏱ Time: %.1f seconds\n🎯 Total Attempts: %d\n⬆️ Next-Level Infections: %d\n➡️  Same-Level Infections: %d\n\n📊 FINAL SCORE: %d points",
		elapsed.Seconds(),
		gameState.stats.Attempts,
		gameState.stats.NextLevelInfections,
		gameState.stats.SameLevelInfections,
		finalScore,
	)
	scoreLabel := widget.NewLabel(scoreText)
	scoreLabel.Alignment = fyne.TextAlignCenter

	// Show infected animals
	var statusCards []fyne.CanvasObject
	for _, animal := range gameState.animals {
		if !animal.Infected {
			continue
		}

		img := loadAnimalImage(animal.GetImagePath(), true)
		
		nameLabel := widget.NewLabel(animal.Name)
		nameLabel.Alignment = fyne.TextAlignCenter

		levelLabel := widget.NewLabel(fmt.Sprintf("Level %d", animal.Level))
		levelLabel.Alignment = fyne.TextAlignCenter

		card := container.NewVBox(img, nameLabel, levelLabel)
		statusCards = append(statusCards, card)
	}

	grid := container.New(layout.NewGridLayout(4), statusCards...)
	scrollable := container.NewScroll(grid)

	playAgainBtn := widget.NewButton("🔄 Play Again", func() {
		animals, maxLevel := LoadAnimalsFromJSON("yellowstone_animals.json")
		redFacts := LoadRedHerringFacts("red_herring_facts.json")
		
		newGameState := &GameState{
			animals:    animals,
			maxLevel:   maxLevel,
			currentDay: 1,
			virus:      &Virus{Modes: []string{"Bite"}, Strength: 1.0},
			stats:      Stats{},
			redFacts:   redFacts,
		}
		
		introScreen := createIntroScreen(myApp, myWindow, newGameState)
		myWindow.SetContent(introScreen)
	})
	playAgainBtn.Importance = widget.HighImportance

	infectedTitle := widget.NewLabel("Infected Animals:")
	infectedTitle.Alignment = fyne.TextAlignCenter
	infectedTitle.TextStyle = fyne.TextStyle{Bold: true}

	content := container.NewBorder(
		container.NewVBox(
			title,
			winLabel,
			scoreLabel,
			playAgainBtn,
			widget.NewSeparator(),
			infectedTitle,
		),
		nil,
		nil,
		nil,
		scrollable,
	)

	return content
}

func main() {
	myApp := app.New()
	myWindow := myApp.NewWindow("🦠 Yellowstone Outbreak: Evolution")
	myWindow.Resize(fyne.NewSize(1200, 800))

	// Load data
	animals, maxLevel := LoadAnimalsFromJSON("yellowstone_animals.json")
	redFacts := LoadRedHerringFacts("red_herring_facts.json")

	if animals == nil {
		dialog.ShowError(fmt.Errorf("Failed to load animals.json"), myWindow)
		return
	}

	// Initialize game state
	gameState := &GameState{
		animals:    animals,
		maxLevel:   maxLevel,
		currentDay: 1,
		virus:      &Virus{Modes: []string{"Bite"}, Strength: 1.0},
		stats:      Stats{},
		redFacts:   redFacts,
	}

	// Show intro screen
	introScreen := createIntroScreen(myApp, myWindow, gameState)
	myWindow.SetContent(introScreen)

	myWindow.ShowAndRun()
}