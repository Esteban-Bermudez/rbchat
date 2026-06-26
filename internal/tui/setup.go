package tui

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"

	"github.com/esteban/rbchat/internal/db"
)

func RunSetup(database *sql.DB) (string, string, error) {
	ctx := context.Background()
	q := db.New(database)

	username, err := q.GetConfig(ctx, "username")
	if err == nil && username != "" {
		team, err := q.GetConfig(ctx, "team")
		if err == nil {
			return username, team, nil
		}
	}

	fmt.Print("Enter your username: ")
	var usernameInput string
	fmt.Scanln(&usernameInput)
	if usernameInput == "" {
		return "", "", fmt.Errorf("username cannot be empty")
	}

	fmt.Println("\nSelect your team:")
	for i, team := range teams {
		fmt.Printf("  %d. %s\n", i+1, team)
	}
	fmt.Print("Enter number (1-" + strconv.Itoa(len(teams)) + "): ")
	var teamChoice int
	fmt.Scanln(&teamChoice)
	if teamChoice < 1 || teamChoice > len(teams) {
		return "", "", fmt.Errorf("invalid team selection")
	}
	selectedTeam := teams[teamChoice-1]

	q.SetConfig(ctx, db.SetConfigParams{Key: "username", Value: usernameInput})
	q.SetConfig(ctx, db.SetConfigParams{Key: "team", Value: selectedTeam})

	return usernameInput, selectedTeam, nil
}
