package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
)

const githubAPI = "https://api.github.com/graphql"

type GraphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type GraphQLResponse struct {
	Data struct {
		User struct {
			ContributionsCollection struct {
				ContributionCalendar struct {
					Weeks []struct {
						ContributionDays []struct {
							Date              string `json:"date"`
							ContributionCount int    `json:"contributionCount"`
						} `json:"contributionDays"`
					} `json:"weeks"`
				} `json:"contributionCalendar"`
			} `json:"contributionsCollection"`
		} `json:"user"`
	} `json:"data"`
}

func colorForCount(count int) string {
	switch {
	case count == 0:
		return "\033[48;5;236m  \033[0m" // no contributions
	case count < 2:
		return "\033[48;5;114m  \033[0m" // light green
	case count < 6:
		return "\033[48;5;34m  \033[0m" // medium green
	case count < 10:
		return "\033[48;5;28m  \033[0m" // vivid green
	default:
		return "\033[48;5;22m  \033[0m" // dark green
	}
}

func main() {
	if err := godotenv.Load(".env"); err != nil {
		log.Println("❌ Can't load the .env file:", err)
	}

	token := os.Getenv("GITHUB_TOKEN")
	username := os.Getenv("GITHUB_USER")
	if token == "" {
		log.Fatal("❌ GITHUB_TOKEN is not defined in the .env file")
	}

	query := `
	query($userName:String!){
	  user(login:$userName){
		contributionsCollection {
		  contributionCalendar {
			weeks {
			  contributionDays {
				date
				contributionCount
			  }
			}
		  }
		}
	  }
	}`

	reqBody := GraphQLRequest{
		Query: query,
		Variables: map[string]interface{}{
			"userName": username,
		},
	}
	jsonData, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", githubAPI, bytes.NewBuffer(jsonData))
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var gqlResp GraphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		panic(err)
	}

	// only the current month
	now := time.Now()
	year, month, _ := now.Date()
	firstDay := time.Date(year, month, 1, 0, 0, 0, 0, time.Local)
	daysInMonth := firstDay.AddDate(0, 1, -1).Day()

	// map with day and number of contributions
	contribMap := make(map[int]int)
	for _, week := range gqlResp.Data.User.ContributionsCollection.ContributionCalendar.Weeks {
		for _, d := range week.ContributionDays {
			t, _ := time.Parse("2006-01-02", d.Date)
			if t.Year() == year && t.Month() == month {
				contribMap[t.Day()] = d.ContributionCount
			}
		}
	}

	fmt.Printf("\nContributions: %s %d\n\n", month, year)
	fmt.Println("Mon Tue Wed Thu Fri Sat Sun")

	startWeekday := int(firstDay.Weekday())
	if startWeekday == 0 {
		startWeekday = 7
	}
	for i := 1; i < startWeekday; i++ {
		fmt.Print("    ")
	}

	for day := 1; day <= daysInMonth; day++ {
		count := contribMap[day]
		block := colorForCount(count)

		fmt.Print(block)

		fmt.Print("  ")

		weekday := (startWeekday + day - 1) % 7
		if weekday == 0 {
			fmt.Println()
		}
	}
}
