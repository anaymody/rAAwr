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

type Mobility string

type Animal struct {
	Name          string   `json:"Name"`
	Level         int      `json:"Level"`
	Mobility      string   `json:"Mobility"`
	Intelligence  int      `json:"Intelligence"`
	Contacts      []string `json:"Contacts"`
	Infected      bool     `json:"Infected"`
	InfectionRate float64  `json:"InfectionRate"`
	Location      string   `json:"Location"`
}

type Virus struct {
	Modes    []string
	Strength float64
}

// -------- JSON Loader --------

func LoadAnimalsFromJSON(filepath string) map[string]*Animal {
	jsonFile, err := ioutil.ReadFile(filepath)
	if err != nil {
		log.Fatalf("❌ Failed to load JSON file: %v", err)
	}

	var data struct {
		Animals []*Animal `json:"animals"`
	}

	err = json.Unmarshal(jsonFile, &data)
	if err != nil {
		log.Fatalf("❌ Failed to parse JSON: %v", err)
	}

	animalMap := make(map[string]*Animal)
	for _, a := range data.Animals {
		animalMap[a.Name] = a
	}

	return animalMap
}

// -------- Infection System --------

func attemptInfection(source *Animal, target *Animal, v *Virus) bool {
	rand.Seed(time.Now().UnixNano())
	chance := target.InfectionRate * v.Strength

	fmt.Printf("\n🔬 Attempting infection: %s → %s\n", source.Name, target.Name)
	fmt.Printf("📈 Chance: %.0f%%\n", chance*100)

	if rand.Float64() < chance {
		target.Infected = true
		fmt.Printf("💥 SUCCESS! %s is now infected.\n", target.Name)
		return true
	}

	fmt.Printf("❌ Failed. %s resisted infection.\n", target.Name)
	return false
}

func printStatus(animals map[string]*Animal) {
	fmt.Println("\n📊 Infection Status:")
	for _, a := range animals {
		status := "😐 Healthy"
		if a.Infected {
			status = "☣️  INFECTED"
		}
		fmt.Printf(" - %-20s : %s\n", a.Name, status)
	}
	fmt.Println()
}

// -------- Player Choice --------

func chooseTarget(player *Animal, animals map[string]*Animal) *Animal {
	reader := bufio.NewReader(os.Stdin)

	var validTargets []string
	fmt.Println("\nWho do you want to infect?")

	i := 1
	for _, contact := range player.Contacts {
		if animals[contact] != nil && !animals[contact].Infected {
			fmt.Printf("%d) %s\n", i, contact)
			validTargets = append(validTargets, contact)
			i++
		}
	}

	fmt.Printf("%d) Skip turn\n", i)

	for {
		fmt.Print("\nEnter choice: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		for idx, val := range validTargets {
			if input == fmt.Sprint(idx+1) || strings.EqualFold(input, val) {
				return animals[val]
			}
		}

		if input == fmt.Sprint(i) {
			return nil
		}

		fmt.Println("❌ Invalid choice — try again.")
	}
}

// -------- Starter Selection --------

func askStarterAnimal(animals map[string]*Animal) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Choose your starting infected animal:")

	i := 1
	for name := range animals {
		fmt.Printf("%d) %s\n", i, name)
		i++
	}

	for {
		fmt.Print("\nEnter animal name: ")
		input, _ := reader.ReadString('\n')
		choice := strings.TrimSpace(input)

		if _, exists := animals[choice]; exists {
			return choice
		}

		fmt.Println("❌ Invalid animal, try again.")
	}
}

// -------- Main Game Loop --------

func main() {
	animals := LoadAnimalsFromJSON("data/yellowstone_animals.json")

	virus := &Virus{Modes: []string{"Bite"}, Strength: 1.0}

	start := askStarterAnimal(animals)
	player := animals[start]
	player.Infected = true

	fmt.Printf("\n🦠 You are now: %s\n", player.Name)

	for turn := 1; turn <= 6; turn++ {
		fmt.Printf("\n===== DAY %d =====\n", turn)

		printStatus(animals)

		target := chooseTarget(player, animals)
		if target != nil {
			attemptInfection(player, target, virus)
		} else {
			fmt.Println("⏸ Skipped turn.")
		}

		time.Sleep(time.Second)
	}

	fmt.Println("\n🏁 Simulation Complete!")
	printStatus(animals)
}
