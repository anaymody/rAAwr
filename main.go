package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
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

var maxLevel int // highest level in the data

// ------------- JSON LOADERS -------------

func LoadAnimalsFromJSON(filepath string) map[string]*Animal {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("❌ Error loading animal file: %v", err)
	}

	var raw map[string][]*Animal
	err = json.Unmarshal(data, &raw)
	if err != nil {
		log.Fatalf("❌ JSON parse error: %v", err)
	}

	result := map[string]*Animal{}
	maxLevel = 0

	for _, group := range raw {
		for _, animal := range group {
			result[animal.Name] = animal
			if animal.Level > maxLevel {
				maxLevel = animal.Level
			}
		}
	}
	return result
}

func LoadRedHerringFacts(filepath string) map[string]RedHerringInfo {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		fmt.Println("⚠ No red herring facts file found.")
		return map[string]RedHerringInfo{}
	}

	var info map[string]RedHerringInfo
	json.Unmarshal(data, &info)
	return info
}

// ------------- TARGET SELECTION LOGIC -------------

// Valid infection targets: same level or next level
func getValidTargets(player *Animal, animals map[string]*Animal) []string {
	valid := []string{}
	for name, a := range animals {
		if a.Infected {
			continue
		}
		if a.Level == player.Level || a.Level == player.Level+1 {
			valid = append(valid, name)
		}
	}
	return valid
}

// ------------- STARTER SELECTION (LEVEL 1 ONLY, RED HERRING CHECK) -------------

func askStarterAnimal(animals map[string]*Animal, redFacts map[string]RedHerringInfo) string {
	reader := bufio.NewReader(os.Stdin)

	levelOne := []string{}
	for name, a := range animals {
		if a.Level == 1 {
			levelOne = append(levelOne, name)
		}
	}

	for {
		fmt.Println("Choose your starting Level 1 animal:\n")
		for i, name := range levelOne {
			fmt.Printf("%d) %s\n", i+1, name)
		}

		fmt.Print("\nEnter a number: ")
		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)

		choice := -1
		fmt.Sscanf(choiceStr, "%d", &choice)

		if choice >= 1 && choice <= len(levelOne) {
			selected := levelOne[choice-1]
			a := animals[selected]

			if a.RedHerring {
				fmt.Println("\n🚫 Cannot start as this animal — RED HERRING.\n")

				if info, ok := redFacts[a.Name]; ok {
					fmt.Printf("🐾 Fun Fact: %s\n", info.FunFact)
					fmt.Printf("📌 Reason: %s\n\n", info.Reason)
				}

				fmt.Println("🔁 Returning to selection...\n")
				continue
			}
			return selected
		}

		fmt.Println("❌ Invalid selection — try again.\n")
	}
}

// ------------- PER TURN: CHOOSE INFECTION TARGET -------------

func chooseTarget(player *Animal, animals map[string]*Animal, redFacts map[string]RedHerringInfo) *Animal {
	reader := bufio.NewReader(os.Stdin)

	for {
		valid := getValidTargets(player, animals)

		fmt.Println("\nWho do you want to infect?")
		if len(valid) == 0 {
			fmt.Println("(No valid targets — skipping day.)")
			return nil
		}

		for i, name := range valid {
			fmt.Printf("%d) %s\n", i+1, name)
		}
		fmt.Printf("%d) Skip turn\n", len(valid)+1)

		fmt.Print("\nEnter a number: ")
		choiceStr, _ := reader.ReadString('\n')
		choiceStr = strings.TrimSpace(choiceStr)

		choice := -1
		fmt.Sscanf(choiceStr, "%d", &choice)

		if choice == len(valid)+1 {
			fmt.Println("⏸ Turn skipped.")
			return nil
		}

		if choice >= 1 && choice <= len(valid) {
			targetName := valid[choice-1]
			target := animals[targetName]

			if target.RedHerring {
				fmt.Println("\n🚫 RED HERRING — cannot infect.\n")

				if info, ok := redFacts[target.Name]; ok {
					fmt.Printf("🐾 Fun Fact: %s\n", info.FunFact)
					fmt.Printf("📌 Reason: %s\n\n", info.Reason)
				}

				fmt.Println("▶ Try again.\n")
				continue
			}

			return target
		}

		fmt.Println("❌ Invalid choice — try again.")
	}
}

// ------------- INFECTION + EVOLUTION + STATS -------------

func attemptInfection(player *Animal, target *Animal, virus *Virus, stats *Stats) (*Animal, bool) {
	rand.Seed(time.Now().UnixNano())
	chance := target.InfectionRate * virus.Strength

	reader := bufio.NewReader(os.Stdin)

	stats.Attempts++

	fmt.Printf("\n🦠 Infection Attempt: %s ➜ %s\n", player.Name, target.Name)
	fmt.Printf("📈 Chance: %.0f%%\n", chance*100)

	if rand.Float64() < chance {
		target.Infected = true
		fmt.Printf("💥 SUCCESS: %s is now infected!\n", target.Name)

		// stats: same-level vs next-level infection
		if target.Level == player.Level {
			stats.SameLevelInfections++
		} else if target.Level == player.Level+1 {
			stats.NextLevelInfections++
		}

		// evolution if higher level host
		if target.Level > player.Level {
			fmt.Printf("\n🔄 EVOLUTION: You now inhabit %s.\n", target.Name)
			fmt.Printf("⬆️ Level Up: %d → %d\n", player.Level, target.Level)

			fmt.Print("\n👉 Press ENTER to continue...")
			reader.ReadString('\n')

			// win condition: reached apex predator
			if target.Level == maxLevel {
				return target, true
			}
			return target, false
		}

		fmt.Print("\n👉 Press ENTER to continue...")
		reader.ReadString('\n')
		return player, false
	}

	fmt.Printf("🛑 FAILED: %s resisted infection.\n", target.Name)
	fmt.Print("\n👉 Press ENTER to continue...")
	reader.ReadString('\n')

	return player, false
}

// ------------- STATUS UI -------------

func printStatus(animals map[string]*Animal) {
	fmt.Println("\n📊 Infection Status:")
	for _, a := range animals {
		state := "😐 Healthy"
		if a.Infected {
			state = "☣ INFECTED"
		}
		fmt.Printf(" - %-22s : %s\n", a.Name, state)
	}
	fmt.Println()
}

// ------------- SCORING SYSTEM -------------

func calculateScore(stats Stats, elapsed time.Duration) int {
	seconds := int(elapsed.Seconds())

	score := 1000
	score += stats.NextLevelInfections * 200 // reward: climbing levels efficiently
	score -= stats.SameLevelInfections * 100 // penalty: wasting infection on same-tier
	score -= stats.Attempts * 10             // penalty: too many tries
	score -= seconds / 2                     // slower runs get penalized a bit

	if score < 0 {
		score = 0
	}
	return score
}

// ------------- MAIN -------------

func main() {
	animals := LoadAnimalsFromJSON("data/yellowstone_animals.json")
	redFacts := LoadRedHerringFacts("data/red_herring_facts.json")

	virus := &Virus{Modes: []string{"Bite"}, Strength: 1.0}
	stats := Stats{
		Attempts:            0,
		SameLevelInfections: 0,
		NextLevelInfections: 0,
		StartTime:           time.Now(),
	}

	start := askStarterAnimal(animals, redFacts)
	player := animals[start]
	player.Infected = true

	fmt.Printf("\n🔥 You start as: %s (Level %d)\n", player.Name, player.Level)
	fmt.Printf("🎯 Goal: Reach Level %d (apex predator) as efficiently as possible.\n", maxLevel)

	for day := 1; day <= 999; day++ {
		elapsed := time.Since(stats.StartTime)
		fmt.Printf("\n======== DAY %d ======== (Elapsed: %.1f seconds)\n", day, elapsed.Seconds())
		printStatus(animals)

		target := chooseTarget(player, animals, redFacts)
		if target != nil {
			var won bool
			player, won = attemptInfection(player, target, virus, &stats)

			if won {
				elapsed := time.Since(stats.StartTime)
				fmt.Printf("\n🏆 YOU WIN! You reached the highest level host: %s (Level %d).\n", player.Name, player.Level)
				fmt.Printf("⏱ Time Taken: %.1f seconds\n", elapsed.Seconds())
				fmt.Printf("🎯 Attempts: %d\n", stats.Attempts)
				fmt.Printf("👍 Next-level infections: %d\n", stats.NextLevelInfections)
				fmt.Printf("👎 Same-level infections: %d\n", stats.SameLevelInfections)

				finalScore := calculateScore(stats, elapsed)
				fmt.Printf("\n📊 FINAL SCORE: %d points\n", finalScore)

				fmt.Println("\n🧾 Infection Summary:")
				printStatus(animals)
				return
			}
		}
	}

	// If somehow loop exits without win
	fmt.Println("\n🏁 SIMULATION ENDED (no apex reached).")
	elapsed := time.Since(stats.StartTime)
	finalScore := calculateScore(stats, elapsed)
	fmt.Printf("⏱ Time: %.1f seconds | Score: %d\n", elapsed.Seconds(), finalScore)
	printStatus(animals)
}
