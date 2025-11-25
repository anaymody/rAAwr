package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"
)

type Mobility string

const (
	Walk  Mobility = "walk"
	Fly   Mobility = "fly"
	Swim  Mobility = "swim"
	Climb Mobility = "climb"
)

type Animal struct {
	Name          string
	Level         int
	Mobility      Mobility
	Intelligence  int
	Contacts      []string
	Infected      bool
	InfectionRate float64
	Location      string
}

type Virus struct {
	Modes    []string
	Strength float64
}

// -------------------- Hardcoded Data --------------------

func LoadYellowstoneAnimals() map[string]*Animal {
	return map[string]*Animal{
		"Grizzly Bear": {
			Name:          "Grizzly Bear",
			Level:         3,
			Mobility:      Walk,
			Intelligence:  7,
			Contacts:      []string{"Wolf", "Tourist", "Bison"},
			InfectionRate: 0.4,
			Location:      "Forest",
		},
		"Wolf": {
			Name:          "Wolf",
			Level:         3,
			Mobility:      Walk,
			Intelligence:  6,
			Contacts:      []string{"Elk", "Bison", "Grizzly Bear"},
			InfectionRate: 0.55,
			Location:      "Mountains",
		},
		"Bison": {
			Name:          "Bison",
			Level:         2,
			Mobility:      Walk,
			Intelligence:  3,
			Contacts:      []string{"Wolf", "Elk"},
			InfectionRate: 0.25,
			Location:      "Grassland",
		},
		"Elk": {
			Name:          "Elk",
			Level:         2,
			Mobility:      Walk,
			Intelligence:  2,
			Contacts:      []string{"Wolf", "Bison"},
			InfectionRate: 0.3,
			Location:      "Valley",
		},
		"Tourist": {
			Name:          "Tourist",
			Level:         1,
			Mobility:      Walk,
			Intelligence:  9,
			Contacts:      []string{"Grizzly Bear", "Wolf"},
			InfectionRate: 0.75,
			Location:      "Campsite",
		},
	}
}

// -------------------- Infection Attempt --------------------

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

// -------------------- Display --------------------

func printStatus(animals map[string]*Animal) {
	fmt.Println("\n📊 Infection Status:")
	for _, a := range animals {
		status := "😐 Healthy"
		if a.Infected {
			status = "☣️  INFECTED"
		}
		fmt.Printf(" - %-15s : %s\n", a.Name, status)
	}
	fmt.Println()
}

// -------------------- Player Choice --------------------

func chooseTarget(player *Animal, animals map[string]*Animal) *Animal {
	reader := bufio.NewReader(os.Stdin)

	validTargets := []string{}
	fmt.Println("\nWho do you want to infect?")

	i := 1
	for _, name := range player.Contacts {
		if animals[name] != nil && !animals[name].Infected {
			fmt.Printf("%d) %s\n", i, name)
			validTargets = append(validTargets, name)
			i++
		}
	}

	fmt.Printf("%d) Skip turn\n", i)

	for {
		fmt.Print("\nEnter choice: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		// convert numeric selection
		for idx, val := range validTargets {
			if input == fmt.Sprint(idx+1) || strings.EqualFold(input, val) {
				return animals[val]
			}
		}

		// skip
		if input == fmt.Sprint(i) || strings.EqualFold(input, "skip") {
			return nil
		}

		fmt.Println("❌ Invalid choice — try again.")
	}
}

// -------------------- Starter Selection --------------------

func askStarterAnimal(animals map[string]*Animal) string {
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Choose your starting infected animal:")

	i := 1
	options := []string{}
	for name := range animals {
		fmt.Printf("%d) %s\n", i, name)
		options = append(options, name)
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

// -------------------- Game Loop --------------------

func main() {
	animals := LoadYellowstoneAnimals()
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
			fmt.Println("⏸ Skipping turn...")
		}

		time.Sleep(time.Second)
	}

	fmt.Println("\n🏁 Simulation Complete!")
	printStatus(animals)
}
